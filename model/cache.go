package model

import (
	"errors"
	"finance/contrib/helper"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
	"github.com/go-redis/redis/v8"
	"sort"
	"time"
)

func CacheRefreshPaymentOfflineBanks() error {

	ex := g.Ex{
		"state": "1",
		"flags": "1",
	}
	res, err := BankCardList(ex)
	if err != nil {
		fmt.Println("BankCardUpdateCache err = ", err)
		return err
	}

	if len(res) == 0 {
		fmt.Println("BankCardUpdateCache len(res) = 0")
		return nil
	}

	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	bkey := meta.Prefix + ":BK:766870294997073616"
	pipe.Unlink(ctx, bkey)
	if len(res) > 0 {

		for k, v := range res {
			bt, err := getBankTypeByCode(bankCodeMap[v.ChannelBankId])
			if err == nil {
				res[k].BanklcardName = bt.ShortName
				res[k].Logo = bt.Logo
			}
		}
		sort.SliceStable(res, func(i, j int) bool {
			if res[i].DailyMaxAmount < res[j].DailyMaxAmount {
				return true
			}

			return false
		})

		s, err := helper.JsonMarshal(res)
		if err != nil {
			return errors.New(helper.FormatErr)
		}

		pipe.Set(ctx, bkey, string(s), 999999*time.Hour)
		pipe.Persist(ctx, bkey)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}
	return nil

	return nil
}

func getBankTypeByCode(bankCode string) (TblBankTypes, error) {

	key := meta.Prefix + ":bank:type:" + bankCode
	data := TblBankTypes{}
	if bankCode == "1047" {
		data.Logo = "https://dl-sg.td22t5f.com/cgpay/VBSP.png"
		data.ShortName = "VBSP"
		return data, nil
	} else if bankCode == "1048" {
		data.Logo = "https://dl-sg.td22t5f.com/cgpay/VDB.png"
		data.ShortName = "VDB"
		return data, nil
	}
	re := meta.MerchantRedis.HMGet(ctx, key, "tr_code", "name_cn", "name_en", "name_vn", "short_name", "swift_code", "alias", "state", "has_otp", "logo")
	if re.Err() != nil {
		if re.Err() == redis.Nil {
			return data, nil
		}

		return data, pushLog(re.Err(), helper.RedisErr)
	}

	if err := re.Scan(&data); err != nil {
		return data, pushLog(err, helper.RedisErr)
	}

	return data, nil
}

func BankTypeUpdateCache() error {

	var data []TblBankTypes
	ex := g.Ex{}
	query, _, _ := dialect.From("tbl_bank_types").Select(coleBankTypes...).Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&data, query)
	if err != nil {
		return err
	}

	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	for _, val := range data {
		value := map[string]interface{}{
			"tr_code":    val.TrCode,
			"name_cn":    val.NameCn,
			"name_en":    val.NameEn,
			"name_vn":    val.NameVn,
			"short_name": val.ShortName,
			"swift_code": val.SwiftCode,
			"alias":      val.Alias,
			"state":      val.State,
			"has_otp":    val.HasOtp,
			"logo":       val.Logo,
		}
		pkey := meta.Prefix + ":bank:type:" + val.BankCode
		pipe.Unlink(ctx, pkey)
		pipe.HMSet(ctx, pkey, value)
		pipe.Persist(ctx, pkey)
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}
	return nil
}

// CachePayment 获取支付方式
func CachePayment(id string) (FPay, error) {

	m := FPay{}
	var cols []string

	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	for _, val := range colPayment {
		cols = append(cols, val.(string))
	}

	pkey := meta.Prefix + ":p:" + id
	// 需要执行的命令
	exists := pipe.Exists(ctx, pkey)
	rs := pipe.HMGet(ctx, pkey, cols...)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return m, err
	}

	if exists.Val() == 0 {
		return m, errors.New(helper.RedisErr)
	}

	err = rs.Scan(&m)
	if err != nil {
		return m, err
	}
	return m, nil
}

// 限制用户存款频率
func cacheDepositProcessing(uid string, now int64) error {

	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	// 检查是否被手动锁定
	manual_lock_key := fmt.Sprintf("%s:finance:mlock:%s", meta.Prefix, uid)
	automatic_lock_key := fmt.Sprintf("%s:finance:alock:%s", meta.Prefix, uid)

	exists := pipe.Exists(ctx, manual_lock_key)

	// 检查是否被自动锁定
	rs := pipe.ZRevRangeWithScores(ctx, automatic_lock_key, 0, -1)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}

	if exists.Val() != 0 {
		return errors.New(helper.NoChannelErr)
	}

	recs, err := rs.Result()
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}

	num := len(recs)
	if num < 10 {
		return nil
	}

	// 十笔 订单 锁定 5 分钟
	if num == 20 && now < int64(recs[0].Score)+5*60 {
		// 最后一笔订单的时间
		return errors.New(helper.EmptyOrder30MinsBlock)
	}

	// 超出10笔 每隔五笔限制24小时
	if num > 10 && num%5 == 0 && now < int64(recs[0].Score)+24*60*60 {
		return errors.New(helper.EmptyOrder5HoursBlock)
	}

	return nil
}

func cacheDepositProcessingInsert(uid, depositId string, now int64) error {

	automatic_lock_key := fmt.Sprintf("%s:finance:alock:%s", meta.Prefix, uid)

	z := redis.Z{
		Score:  float64(now),
		Member: depositId,
	}
	return meta.MerchantRedis.ZAdd(ctx, automatic_lock_key, &z).Err()
}

// CacheDepositProcessingRem 清除未未成功的订单计数
func CacheDepositProcessingRem(uid string) error {

	automatic_lock_key := fmt.Sprintf("%s:finance:alock:%s", meta.Prefix, uid)
	return meta.MerchantRedis.Unlink(ctx, automatic_lock_key).Err()
}

func withLock(id string) error {

	val := fmt.Sprintf("%s:%s%s", meta.Prefix, defaultRedisKeyPrefix, id)
	ok, err := meta.MerchantRedis.SetNX(ctx, val, "1", 120*time.Second).Result()
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}
	if !ok {
		return errors.New(helper.RequestBusy)
	}

	return nil
}
