package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"finance/contrib/validator"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
	"strconv"
)

type Bankcard_t struct {
	Id                string `db:"id" json:"id" json:"id"`
	ChannelBankId     string `db:"channel_bank_id" json:"bank_id"`                        // t_channel_bank的id
	BanklcardName     string `db:"banklcard_name" json:"banklcard_name"`                  // 银行名称
	BanklcardNo       string `db:"banklcard_no" json:"banklcard_no"`                      // 银行卡号
	AccountName       string `db:"account_name" json:"account_name"`                      // 持卡人姓名
	BankcardAddr      string `db:"bankcard_addr" json:"bankcard_addr"`                    // 开户行地址
	State             string `db:"state" json:"state"`                                    // 状态：0 关闭  1 开启
	Remark            string `db:"remark" json:"remark"`                                  // 备注
	Prefix            string `db:"prefix" json:"prefix"`                                  // 商户前缀
	DailyMaxAmount    string `db:"daily_max_amount" json:"daily_max_amount"`              // 当天最大收款限额
	DailyFinishAmount string `db:"daily_finish_amount" json:"daily_finish_amount"`        // 当天已收款总额
	Flags             string `db:"flags" json:"flags"`                                    // 1转卡2转账
	Logo              string `db:"-" json:"logo"`                                         //logo
	VipList           string `db:"vip_list" json:"vip_list"`                              //vip等级
	Fmin              string `db:"fmin" json:"fmin"`                                      //最小金额
	Fmax              string `db:"fmax" json:"fmax"`                                      //最大金额
	AmountList        string `db:"amount_list" json:"amount_list"`                        //金额列表
	Discount          string `db:"discount" json:"discount"`                              //优惠
	IsZone            int    `db:"is_zone" json:"is_zone"`                                // 0 不是区间 1是区间
	IsFast            int    `db:"is_fast" json:"is_fast"`                                // 0 不是快捷 1是快捷
	Cid               int64  `db:"cid" json:"cid"`                                        //1:QR Banking 2:MomoPay 3:ZaloPay 4:ViettelPay 5:Thẻ Cào 6:Offline 7:USDT
	CreatedAt         int64  `db:"created_at" json:"created_at" redis:"created_at"`       //创建时间
	CreatedUID        string `db:"created_uid" json:"created_uid" redis:"created_uid"`    //创建人的ID
	CreatedName       string `db:"created_name" json:"created_name" redis:"created_name"` //创建人的名字
	UpdatedAt         int64  `db:"updated_at" json:"updated_at" redis:"updated_at"`       //操作时间
	UpdatedUID        string `db:"updated_uid" json:"updated_uid" redis:"updated_uid"`    //操作人的ID
	UpdatedName       string `db:"updated_name" json:"updated_name" redis:"updated_name"` //操作人的名字
	Seq               int    `db:"seq" json:"seq" redis:"seq"`                            //排序
	PaymentName       string `db:"payment_name" json:"payment_name" redis:"payment_name"` //前台展示名称
}

// BankCardList 银行卡列表
func BankCardList(ex g.Ex, vip string) ([]Bankcard_t, error) {

	var data []Bankcard_t
	query, _, _ := dialect.From("f2_bankcards").Select(colsBankCard...).Where(ex).ToSQL()
	fmt.Println(query)
	err := meta.MerchantDB.Select(&data, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	return data, nil
}

func BankCardByCol(val string) (Bankcard_t, error) {

	var bc Bankcard_t
	ex := g.Ex{
		"banklcard_no": val,
	}
	query, _, _ := dialect.From("f2_bankcards").Select(colsBankCard...).Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&bc, query)
	if err != nil && err != sql.ErrNoRows {
		return bc, pushLog(err, helper.DBErr)
	}

	if err == sql.ErrNoRows {
		return bc, errors.New(helper.RecordNotExistErr)
	}

	return bc, nil
}

func BankCardInsert(recs Bankcard_t, code, adminName string) error {

	if !validator.CheckStringVName(recs.AccountName) {
		return errors.New(helper.ParamErr)
	}

	recs.Prefix = meta.Prefix

	query, _, _ := dialect.Insert("f2_bankcards").Rows(recs).ToSQL()

	fmt.Println("BankCardInsert query = ", query)
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}
	BankCardUpdateCache()

	contentLog := fmt.Sprintf("渠道管理-线下银行卡-新增:后台账号:%s【id:%s,银行卡名称:%s,卡号:%s,姓名:%s,用途:%s,当日最大入款金额:%s",
		adminName, recs.Id, recs.BanklcardName, recs.BanklcardNo, recs.AccountName, "收款", recs.DailyMaxAmount)
	AdminLogInsert(ChannelModel, contentLog, InsertOp, adminName)

	return nil
}

func BankCardUpdateCache() error {

	key := meta.Prefix + ":offline:bank:"
	ex := g.Ex{
		"state": "1",
	}
	res, err := BankCardList(ex, "")
	if err != nil {
		fmt.Println("BankCardUpdateCache err = ", err)
		return err
	}

	if len(res) == 0 {
		fmt.Println("BankCardUpdateCache len(res) = 0")
		meta.MerchantRedis.Unlink(ctx, key).Err()
		return nil
	}
	meta.MerchantRedis.Unlink(ctx, key+"6", key+"2", key+"4", key+"3", key+"7")

	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	for _, v := range res {
		val, err := helper.JsonMarshal(v)
		if err != nil {
			continue
		}
		pipe.LPush(ctx, fmt.Sprintf(`%s%d`, key, v.Cid), string(val))
		value := map[string]interface{}{
			"account":         v.AccountName,
			"cardno":          v.BanklcardNo,
			"name":            v.BanklcardName,
			"cid":             v.Cid,
			"channel_bank_id": v.ChannelBankId,
		}
		vkey := meta.Prefix + ":offline:bankid:" + v.Id
		pipe.HMSet(ctx, vkey, value)
		pipe.Persist(ctx, vkey)

	}
	pipe.Persist(ctx, key)

	_, err = pipe.Exec(ctx)
	if err != nil {
		fmt.Println("BankCardUpdateCache pipe.Exec = ", err)
		return errors.New(helper.RedisErr)
	}

	CacheRefreshOfflinePaymentBanks("766870294997073617", 2, "1")
	CacheRefreshOfflinePaymentBanks("766870294997073618", 2, "2")
	CacheRefreshOfflinePaymentBanks("766870294997073619", 4, "1")
	CacheRefreshOfflinePaymentBanks("766870294997073620", 4, "2")
	CacheRefreshOfflinePaymentBanks("766870294997073621", 3, "1")
	CacheRefreshOfflinePaymentBanks("766870294997073616", 6, "2")

	return nil
}

func BankCardByID(id string) (Bankcard_t, error) {

	var bc Bankcard_t
	ex := g.Ex{
		"id": id,
	}
	query, _, _ := dialect.From("f2_bankcards").Select(colsBankCard...).Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&bc, query)
	if err != nil && err != sql.ErrNoRows {
		return bc, pushLog(err, helper.DBErr)
	}

	if err == sql.ErrNoRows {
		return bc, errors.New(helper.BankCardNotExist)
	}

	return bc, nil
}

func BankCardUpdate(id string, record g.Record) error {

	ex := g.Ex{
		"id": id,
	}
	query, _, _ := dialect.Update("f2_bankcards").Set(record).Where(ex).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}
	BankCardUpdateCache()
	return nil
}

// BankCardDelete 删除银行卡
func BankCardDelete(id, adminName string) error {

	BankCard, err := BankCardByID(id)
	if err != nil {
		return err
	}
	if BankCard.State == "1" {
		return errors.New(helper.OnlineCateDeleteErr)
	}

	ex := g.Ex{
		"id": id,
	}
	query, _, _ := dialect.Delete("f2_bankcards").Where(ex).ToSQL()
	_, err = meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}
	BankCardUpdateCache()

	contentLog := fmt.Sprintf("渠道管理-线下银行卡-新增:后台账号:%s【id:%s,银行卡名称:%s,卡号:%s,姓名:%s,用途:%s,当日最大入款金额:%s】",
		adminName, id, BankCard.BanklcardName, BankCard.BanklcardNo, BankCard.AccountName, "收款", BankCard.DailyMaxAmount)
	AdminLogInsert(ChannelModel, contentLog, DeleteOp, adminName)

	return nil
}

func BankCardBackendById(bid string) (Bankcard_t, error) {

	bc := Bankcard_t{
		Id: bid,
	}
	key := meta.Prefix + ":offline:bankid:" + bid
	re := meta.MerchantRedis.HMGet(ctx, key, "account", "cardno", "name", "cid", "channel_bank_id")
	if re.Err() != nil {
		return bc, errors.New(helper.RecordNotExistErr)
	}
	scope := re.Val()
	if account, ok := scope[0].(string); !ok {
		return bc, errors.New(helper.TunnelMinLimitErr)
	} else {
		bc.AccountName = account
	}

	if cardno, ok := scope[1].(string); !ok {
		return bc, errors.New(helper.TunnelMaxLimitErr)
	} else {
		bc.BanklcardNo = cardno
	}

	if cardname, ok := scope[2].(string); !ok {
		return bc, errors.New(helper.TunnelMaxLimitErr)
	} else {
		bc.BanklcardName = cardname
	}

	if cid, ok := scope[3].(string); !ok {
		return bc, errors.New(helper.TunnelMaxLimitErr)
	} else {
		cid_int, _ := strconv.ParseInt(cid, 10, 64)
		bc.Cid = cid_int
	}

	if channelBankId, ok := scope[4].(string); !ok {
		return bc, errors.New(helper.TunnelMaxLimitErr)
	} else {
		bc.ChannelBankId = channelBankId
	}

	return bc, nil
}

func BankCardBackend(pid string) (Bankcard_t, error) {

	var bc Bankcard_t
	var (
		cid   int
		flags string
	)
	switch pid {
	case "766870294997073617":
		cid = 2
		flags = "1"
	case "766870294997073618":
		cid = 2
		flags = "2"
	case "766870294997073619":
		cid = 4
		flags = "1"
	case "766870294997073620":
		cid = 4
		flags = "2"
	case "766870294997073621":
		cid = 3
		flags = "1"
	case "766870294997073616":
		cid = 6
		flags = "2"
	}

	ex := g.Ex{
		"cid":   cid,
		"flags": flags,
		"state": 1,
	}
	query, _, _ := dialect.From("f2_bankcards").Select(colsBankCard...).Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&bc, query)
	if err != nil && err != sql.ErrNoRows {
		return bc, pushLog(err, helper.DBErr)
	}

	if err == sql.ErrNoRows {
		return bc, errors.New(helper.BankCardNotExist)
	}
	return bc, nil
}
