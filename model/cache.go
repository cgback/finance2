package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
	"github.com/go-redis/redis/v8"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"
	"sort"
	"strconv"
	"strings"
	"time"
)

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

	for _, val := range colsPayment {
		cols = append(cols, val.(string))
	}

	pkey := meta.Prefix + ":f:p:" + id
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

// depositLock 锁定充值订单 防止并发多充钱
func depositLock(id string) error {

	key := fmt.Sprintf(depositOrderLockKey, id)
	return Lock(key)
}

// depositUnLock 解锁充值订单
func depositUnLock(id string) {
	key := fmt.Sprintf(depositOrderLockKey, id)
	Unlock(key)
}

func CacheRefreshPayment(id string) error {

	val, err := ChanByID(id)
	if err != nil {
		return err
	}

	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	value := map[string]interface{}{
		"id":           val.ID,
		"cate_id":      val.CateID,
		"channel_id":   val.ChannelID,
		"payment_name": val.PaymentName,
		"comment":      val.Comment,
		"created_at":   val.CreatedAt,
		"et":           val.Et,
		"fmax":         val.Fmax,
		"fmin":         val.Fmin,
		"sort":         val.Sort,
		"st":           val.St,
		"state":        val.State,
		"amount_list":  val.AmountList,
		"discount":     val.Discount,
		"is_zone":      val.IsZone,
		"is_fast":      val.IsFast,
		"h5_img":       val.H5Img,
		"web_img":      val.WebImg,
		"app_img":      val.AppImg,
	}
	pkey := meta.Prefix + ":f:p:" + val.ID
	pipe.Unlink(ctx, pkey)
	pipe.HMSet(ctx, pkey, value)
	pipe.Persist(ctx, pkey)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}

	CacheRefreshLevel()
	return nil
}

// 自己的银行
func CacheRefreshOfflinePaymentBanks(id string, cid int64, flags string) error {

	ex := g.Ex{
		"state": "1",
		"flags": flags,
		"cid":   cid,
	}
	res, err := BankCardList(ex, "")
	if err != nil {
		fmt.Println("BankCardUpdateCache err = ", err)
		return err
	}

	if len(res) == 0 {
		fmt.Println("BankCardUpdateCache len(res) = 0")
		//关闭这个渠道的状态
		ChannelSet(id, "0", "0", "")
		return nil
	}

	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	bkey := meta.Prefix + ":BK:" + id
	pipe.Unlink(ctx, bkey)
	if len(res) > 0 {

		for k, v := range res {
			if v.ChannelBankId != "" {
				bt, err := getBankTypeByCode(bankCodeMap[v.ChannelBankId])
				if err == nil {
					res[k].BanklcardName = bt.ShortName
					res[k].Logo = bt.Logo
				}
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
}

// 三方的通道银行
func CacheRefreshPaymentBanks(id string) error {

	return nil
}

//
//func Create(level string) {
//
//	var (
//		cIds       []string
//		paymentIds []string
//		tunnels    []Tunnel_t
//		tunnelSort []Tunnel_t
//		payments   []Payment_t
//	)
//
//	fmt.Println("Create p:" + level)
//	//删除key
//	meta.MerchantRedis.Unlink(ctx, meta.Prefix+":p:"+level).Result()
//
//	tunneldataTemp, err := meta.MerchantRedis.Get(ctx, meta.Prefix+":tunnel:All").Bytes()
//	if err != nil {
//		fmt.Println("tunnel:All = ", err.Error())
//		return
//	}
//
//	helper.JsonUnmarshal(tunneldataTemp, &tunnelSort)
//	fmt.Println("JsonUnmarshal tunnelSort = ", tunnelSort)
//
//	ex := g.Ex{
//		"id":     paymentIds,
//		"state":  "1",
//		"prefix": meta.Prefix,
//	}
//	query, _, _ := dialect.From("f2_payment").Select(colPayment...).Where(ex).ToSQL()
//	queryIn, args, err := sqlx.In(query)
//	if err != nil {
//		fmt.Println("2", err.Error())
//		return
//	}
//
//	err = meta.MerchantDB.Select(&payments, queryIn, args...)
//	if err != nil {
//		fmt.Println("3", err.Error())
//		return
//	}
//
//	for _, val := range payments {
//		if level == "1" {
//			CacheRefreshPaymentBanks(val.ID)
//		}
//		cIds = append(cIds, val.ChannelID)
//	}
//
//	ex = g.Ex{
//		"id":     cIds,
//		"prefix": meta.Prefix,
//	}
//	query, _, _ = dialect.From("f2_channel_type").Select(colsChannelType...).Where(ex).ToSQL()
//	queryIn, args, err = sqlx.In(query)
//	if err != nil {
//		fmt.Println("4", err.Error())
//		return
//	}
//
//	err = meta.MerchantDB.Select(&tunnels, queryIn, args...)
//	if err != nil {
//		fmt.Println("5", err.Error())
//		return
//	}
//
//	pipe := meta.MerchantRedis.TxPipeline()
//
//	for _, val := range payments {
//		pipe.Unlink(ctx, meta.Prefix+":p:"+val.ID)
//		pipe.Unlink(ctx, meta.Prefix+":p:"+level+":"+val.ChannelID)
//	}
//
//	for _, val := range tunnels {
//		value, _ := helper.JsonMarshal(val)
//		vv := new(redis.Z)
//
//		vv.Member = string(value)
//		for _, v := range tunnelSort {
//
//			if val.ID == v.ID {
//				vv.Score = float64(v.Sort)
//			}
//
//		}
//		pipe.ZAdd(ctx, meta.Prefix+":p:"+level, vv)
//		vv = nil
//	}
//
//	pipe.Persist(ctx, meta.Prefix+":p:"+level)
//
//	for _, val := range payments {
//
//		value := map[string]interface{}{
//			"id":           val.ID,
//			"cate_id":      val.CateID,
//			"channel_id":   val.ChannelID,
//			"fmax":         val.Fmax,
//			"fmin":         val.Fmin,
//			"amount_list":  val.AmountList,
//			"et":           val.Et,
//			"st":           val.St,
//			"payment_name": val.PaymentName,
//			"created_at":   val.CreatedAt,
//
//			"state":   val.State,
//			"sort":    val.Sort,
//			"comment": val.Comment,
//		}
//		pipe.LPush(ctx, meta.Prefix+":p:"+level+":"+val.ChannelID, val.ID)
//		pipe.HMSet(ctx, meta.Prefix+":p:"+val.ID, value)
//		pipe.Persist(ctx, meta.Prefix+":p:"+val.ID)
//	}
//
//	_, err = pipe.Exec(ctx)
//	pipe.Close()
//
//	fmt.Println("err = ", err)
//	fmt.Println("tunnels = ", tunnels)
//	fmt.Println("payments = ", payments)
//	fmt.Println("paymentIds = ", paymentIds)
//}

func CacheRefreshLevel() {

	var (
		tunnels  []ChannelType
		payments []Payment_t
	)
	levelMap := map[string][]Payment_t{}
	channelTypeMap := map[string]ChannelType{}

	ex := g.Ex{
		"state":      "1",
		"channel_id": g.Op{"neq": "101"},
		"id":         g.Op{"neq": "133221087319615487"},
	}
	query, _, _ := dialect.From("f2_payment").Select(colPayment...).Where(ex).Order(g.C("sort").Asc()).ToSQL()
	err := meta.MerchantDB.Select(&payments, query)
	if err != nil && err != sql.ErrNoRows {
		return
	}

	//把所有通道按等级划分
	for _, val := range payments {
		vipList := strings.Split(val.VipList, ",")
		for _, level := range vipList {
			if value, exists := levelMap[level]; !exists {
				levelMap[level] = []Payment_t{val}
			} else {
				value = append(value, val)
				levelMap[level] = value
			}
			meta.MerchantRedis.Unlink(ctx, meta.Prefix+":f:c:p:"+level+":"+val.ChannelID).Err()
		}
	}

	ex = g.Ex{
		"state": 1,
	}
	query, _, _ = dialect.From("f2_channel_type").Select(colsChannelType...).Where(ex).Order(g.C("sort").Asc()).ToSQL()
	err = meta.MerchantDB.Select(&tunnels, query)
	if err != nil && err != sql.ErrNoRows {
		return
	}

	pipe := meta.MerchantRedis.TxPipeline()

	for _, val := range tunnels {
		channelTypeMap[val.ID] = val
		value := map[string]interface{}{
			"id":    val.ID,
			"name":  val.Name,
			"sort":  val.Sort,
			"state": val.State,
			"alias": val.Alias,
		}
		pipe.HMSet(ctx, meta.Prefix+":f:c:"+val.ID, value)
		pipe.Persist(ctx, meta.Prefix+":f:c:"+val.ID)
	}

	for _, val := range payments {

		value := map[string]interface{}{
			"id":           val.ID,
			"cate_id":      val.CateID,
			"channel_id":   val.ChannelID,
			"fmax":         val.Fmax,
			"fmin":         val.Fmin,
			"amount_list":  val.AmountList,
			"et":           val.Et,
			"st":           val.St,
			"payment_name": val.PaymentName,
			"created_at":   val.CreatedAt,
			"state":        val.State,
			"sort":         val.Sort,
			"comment":      val.Comment,
			"name":         val.Name,
			"is_zone":      val.IsZone,
			"is_fast":      val.IsFast,
			"flag":         val.Flag,
			"web_img":      val.WebImg,
			"app_img":      val.AppImg,
			"h5_img":       val.H5Img,
		}

		// 這邊寫入chnnel資料
		pipe.HMSet(ctx, meta.Prefix+":f:p:"+val.ID, value)
		pipe.Persist(ctx, meta.Prefix+":f:p:"+val.ID)

	}
	for level, val := range levelMap {
		var pidsTemp []string
		cidMap := map[string]struct{}{}
		cidsTemp := map[string]string{}
		for _, val2 := range val {
			pidsTemp = append(pidsTemp, val2.ID)
			cidsTemp[val2.ChannelID+"_"+val2.ID] = val2.ID
		}
		var cids []string
		for cidPid, pid := range cidsTemp {
			cs := strings.Split(cidPid, "_")
			cid := cs[0]
			if _, ok := cidMap[cid]; !ok {
				cids = append(cids, cid)
				cidMap[cid] = struct{}{}
			}
			pipe.LPush(ctx, meta.Prefix+":f:c:p:"+level+":"+cid, pid)
			pipe.Persist(ctx, meta.Prefix+":f:c:p:"+level+":"+cid)
		}
		if len(pidsTemp) > 0 {
			pipe.Set(ctx, meta.Prefix+":f:p:lvl:"+level, strings.Join(pidsTemp, ","), 1*time.Hour)
			pipe.Persist(ctx, meta.Prefix+":f:p:lvl:"+level)
		}
		if len(cids) > 0 {
			pipe.Set(ctx, meta.Prefix+":f:c:lvl:"+level, strings.Join(cids, ","), 1*time.Hour)
			pipe.Persist(ctx, meta.Prefix+":f:c:lvl:"+level)
		}

	}

	_, err = pipe.Exec(ctx)
	pipe.Close()

	fmt.Println("err = ", err)
	fmt.Println("tunnels = ", tunnels)
	fmt.Println("payments = ", payments)
}

func cateToRedis() error {

	var a = &fastjson.Arena{}

	var cate []Category
	ex := g.Ex{
		"prefix": meta.Prefix,
	}
	query, _, _ := dialect.From("f2_category").Select("*").Where(ex).Order(g.C("id").Asc()).ToSQL()
	err := meta.MerchantDB.Select(&cate, query)

	if err != nil || len(cate) < 1 {
		return err
	}

	obj := a.NewObject()

	for _, v := range cate {
		val := a.NewString(v.Name)

		obj.Set(v.ID, val)
	}

	b := obj.String()

	key := meta.Prefix + ":f:category"
	err = meta.MerchantRedis.Set(ctx, key, b, 0).Err()
	return err
}

func Tunnel(fctx *fasthttp.RequestCtx, id string) (string, error) {

	a := &fastjson.Arena{}

	u, err := MemberCache(fctx)
	if err != nil {
		return "", err
	}
	key := fmt.Sprintf("%s:f:c:p:%d:%s", meta.Prefix, u.Level, id)
	fmt.Println("key:", key)
	lastDepositPaymentKey := fmt.Sprintf("%s:uldp:%s", meta.Prefix, u.Username)
	var lastDepositPayment string

	paymentIds, err := meta.MerchantRedis.LRange(ctx, key, 0, -1).Result() // 59000000000000001
	if err != nil {
		fmt.Println("SMembers = ", err.Error())
		return "[]", nil
	}
	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	// for disable usdt tunnel with id 779402438062874465
	removeIndex := -1
	for i, v := range paymentIds {
		if v == "779402438062874465" {
			removeIndex = i
		}
	}
	if removeIndex != -1 {
		copy(paymentIds[removeIndex:], paymentIds[removeIndex+1:])
		paymentIds = paymentIds[:len(paymentIds)-1]
	} //

	ll := len(paymentIds)
	rs := make([]*redis.SliceCmd, ll)
	bk := make([]*redis.StringCmd, ll)

	exists := pipe.Exists(ctx, fmt.Sprintf("%s:DL:%s", meta.Prefix, u.UID))
	for i, v := range paymentIds {
		fmt.Println("paymentIds key:", meta.Prefix+":f:p:"+v)
		// 這邊讀取支付相關資料
		rs[i] = pipe.HMGet(ctx, meta.Prefix+":f:p:"+v, "id", "fmin", "discount", "fmax", "et", "st", "amount_list", "payment_name", "sort", "name", "is_zone", "is_fast", "flag", "web_img", "h5_img", "app_img") // 應該是這邊可以抓到資料
		bk[i] = pipe.Get(ctx, meta.Prefix+":BK:"+v)
	}
	exists2 := pipe.Exists(ctx, lastDepositPaymentKey)

	pipe.Exec(ctx)

	// 如果会员被锁定不返回渠道
	if exists.Val() != 0 {
		return "[]", pushLog(err, helper.RedisErr)
	}
	if exists2.Val() == 1 {
		lastDeposit := meta.MerchantRedis.Get(ctx, lastDepositPaymentKey)
		lastDepositPayment, _ = lastDeposit.Result()
	} else if u.FirstDepositAt > 0 {
		d, _ := depositLast(u.UID)
		lastDepositPayment = d.ChannelID
		meta.MerchantRedis.Set(ctx, lastDepositPaymentKey, lastDepositPayment, 100*time.Hour).Err()
	}

	arr := a.NewArray()

	for i := 0; i < ll; i++ {

		var (
			m Payment_t
		)
		if err = rs[i].Scan(&m); err != nil {
			return "", pushLog(err, helper.RedisErr)
		}
		fmt.Println("paymentIds key:", m.ID, m.Sort)

		if m.ID == "779402438062874465" {
			continue
		}

		obj := fastjson.MustParse(`{"id":"0","bank":[], "fmin":"0","fmax":"0", "amount_list": "","sort":"0","payment_name":"","discount":"0","name":"","is_zone":"0","is_fast":"0","flag":"1","web_img":"","h5_img":"","app_img":""}`)
		obj.Set("id", fastjson.MustParse(fmt.Sprintf(`"%s"`, m.ID)))
		obj.Set("fmin", fastjson.MustParse(fmt.Sprintf(`"%s"`, m.Fmin)))
		obj.Set("fmax", fastjson.MustParse(fmt.Sprintf(`"%s"`, m.Fmax)))
		obj.Set("sort", fastjson.MustParse(fmt.Sprintf(`"%s"`, m.Sort)))
		obj.Set("payment_name", fastjson.MustParse(fmt.Sprintf(`"%s"`, m.PaymentName)))
		obj.Set("amount_list", fastjson.MustParse(fmt.Sprintf(`"%s"`, m.AmountList)))
		obj.Set("discount", fastjson.MustParse(fmt.Sprintf(`"%s"`, m.Discount)))
		obj.Set("name", fastjson.MustParse(fmt.Sprintf(`"%s"`, m.Name)))
		obj.Set("is_zone", fastjson.MustParse(fmt.Sprintf(`"%s"`, m.IsZone)))
		obj.Set("is_fast", fastjson.MustParse(fmt.Sprintf(`"%s"`, m.IsFast)))
		obj.Set("flag", fastjson.MustParse(fmt.Sprintf(`"%s"`, m.Flag)))
		obj.Set("web_img", fastjson.MustParse(fmt.Sprintf(`"%s"`, m.WebImg)))
		obj.Set("h5_img", fastjson.MustParse(fmt.Sprintf(`"%s"`, m.H5Img)))
		obj.Set("app_img", fastjson.MustParse(fmt.Sprintf(`"%s"`, m.AppImg)))

		if m.ID == lastDepositPayment {
			obj.Set("is_last_success", fastjson.MustParse("1"))
		} else {
			obj.Set("is_last_success", fastjson.MustParse("0"))
		}
		if m.ID == "779402438062874469" {
			//是usdt的话要添加usdt的账号
			usdtRate, err := UsdtInfo()
			if err == nil {

				obj.Set("rate", fastjson.MustParse(fmt.Sprintf(`"%s"`, usdtRate["deposit_usdt_rate"])))
				usdtAccountsKey := meta.Prefix + ":offline:usdt:one"
				usdtid, err := meta.MerchantRedis.Get(ctx, usdtAccountsKey).Result()
				idkey := fmt.Sprintf("%s:%s", usdtAccountsKey, usdtid)
				fields := []string{"qr_img", "protocol", "min_amount", "max_amount", "wallet_addr"}
				result, err := meta.MerchantRedis.HMGet(ctx, idkey, fields...).Result()
				if err != nil {
					fmt.Println("Error:", err)
				}

				for j, field := range result {
					if field != nil {
						obj.Set(fields[j], fastjson.MustParse(fmt.Sprintf(`"%s"`, field.(string))))
					}
					if fields[j] == "max_amount" {
						obj.Set("fmax", fastjson.MustParse(fmt.Sprintf(`"%s"`, field.(string))))
					}
					if fields[j] == "min_amount" {
						obj.Set("fmin", fastjson.MustParse(fmt.Sprintf(`"%s"`, field.(string))))
					}
				}
			}
		}

		banks := bk[i].Val()
		if len(banks) > 0 {
			obj.Set("bank", fastjson.MustParse(banks))
		}

		arr.SetArrayItem(i, obj)
		obj = nil
		if m.ID == "766870294997073616" {
			break
		}
	}
	str := arr.String()

	return str, nil
}

type CateData struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	IsLastSuccess int    `json:"is_last_success"`
	Sort          int    `json:"sort"`
	PromoDiscount string `json:"promo_discount"`
}

func Cate(fctx *fasthttp.RequestCtx) ([]CateData, error) {

	data := []CateData{}

	m, err := MemberCache(fctx)
	if err != nil {
		return data, err
	}
	var lastDepositChannel string
	key := fmt.Sprintf("%s:f:c:lvl:%d", meta.Prefix, m.Level)
	lastDepositKey := fmt.Sprintf("%s:uld:%s", meta.Prefix, m.Username)
	pipe := meta.MerchantRedis.Pipeline()
	exists := pipe.Exists(ctx, fmt.Sprintf("%s:DL:%s", meta.Prefix, m.UID))

	recs_temp := pipe.Get(ctx, key)
	exists2 := pipe.Exists(ctx, lastDepositKey)

	_, err = pipe.Exec(ctx)
	pipe.Close()
	if err != nil {
		return data, pushLog(err, helper.RedisErr)
	}

	recs_result, err := recs_temp.Result()
	if err != nil {
		return data, nil
	}
	// 如果会员被锁定不返回通道
	if exists.Val() != 0 {
		return data, nil
	}
	if exists2.Val() == 1 {
		lastDeposit := meta.MerchantRedis.Get(ctx, lastDepositKey)
		lastDepositChannel, _ = lastDeposit.Result()
	} else if m.FirstDepositAt > 0 {
		d, _ := depositLast(m.UID)
		lastDepositChannel = d.ChannelID
		meta.MerchantRedis.Set(ctx, lastDepositKey, lastDepositChannel, 100*time.Hour).Err()
	}

	recs := strings.Split(recs_result, ",")
	for _, value := range recs {

		val := CateData{}
		re := meta.MerchantRedis.HMGet(ctx, meta.Prefix+":p:c:t:"+value, "id", "name", "sort", "promo_discount")
		scope := re.Val()
		if id, ok := scope[0].(string); !ok {
			fmt.Println("scope:", scope)
			continue
		} else {
			if lastDepositChannel != "" && string(id) == lastDepositChannel {
				val.IsLastSuccess = 1
			} else {
				val.IsLastSuccess = 0
			}
			val.Id = id
		}
		if name, ok := scope[1].(string); !ok {
			continue
		} else {
			val.Name = name
		}
		if sort, ok := scope[2].(string); !ok {
			continue
		} else {
			sortTemp, err := strconv.Atoi(sort)
			if err == nil {
				val.Sort = sortTemp
			}
		}

		if promoDiscount, ok := scope[3].(string); !ok {
			continue
		} else {
			val.PromoDiscount = promoDiscount
		}

		data = append(data, val)
	}
	sort.Slice(data, func(i, j int) bool { return data[i].Sort < data[j].Sort })

	return data, nil
}

// WithdrawLock 锁定提款订单
// 订单因为外部因素(接口)导致的状态流转应该加锁
func WithdrawLock(id string) error {

	key := fmt.Sprintf(withdrawOrderLockKey, id)
	err := Lock(key)
	if err != nil {
		return err
	}

	return nil
}

// WithdrawUnLock 解锁提款订单
func WithdrawUnLock(id string) {
	Unlock(fmt.Sprintf(withdrawOrderLockKey, id))
}

func CateListRedis() string {

	res, err := meta.MerchantRedis.Get(ctx, meta.Prefix+":f:category").Result()
	if err == redis.Nil || err != nil {
		return "{}"
	}

	return res
}
