package model

import (
	"errors"
	"finance/contrib/helper"
	"fmt"
	"strconv"

	g "github.com/doug-martin/goqu/v9"
	"github.com/go-redis/redis/v8"
)

type Receive struct {
	ID   string `db:"id" json:"id" rule:"none"`      // 主键ID
	Name string `db:"name" json:"name" rule:"aname"` // 用户名
}

type ReceiveData struct {
	D         []Receive `json:"d" db:"d"`
	EnableMod bool      `json:"enable_mod" db:"enable_mod"`
}

// 返水风控审核人员的UID
func GetRisksUID() (string, error) {

	maxKey := fmt.Sprintf("%s:risk:maxreceivenum", meta.Prefix)
	// 查询最大接单数量
	max, err := meta.MerchantRedis.Get(ctx, maxKey).Uint64()
	if err != nil && err != redis.Nil {
		return "0", pushLog(err, helper.RedisErr)
	}

	// 如果最大接单数量小于等于0则直接返回
	if max <= 0 {
		return "0", errors.New("max acceptable order quality less or equal to 0")
	}

	risksKey := fmt.Sprintf("%s:risk:receive", meta.Prefix)
	// 查询在自动派单列表中的总人数
	c, err := meta.MerchantRedis.LLen(ctx, risksKey).Result()
	if err != nil {
		return "0", pushLog(err, helper.RedisErr)
	}

	for i := int64(0); i < c; i++ {
		uid, err := meta.MerchantRedis.RPopLPush(ctx, risksKey, risksKey).Result()
		if err != nil && err != redis.Nil {
			return "0", pushLog(err, helper.RedisErr)
		}

		// 查询结果可能是redis.Nil
		if uid == "" {
			continue
		}

		key := fmt.Sprintf("%s:risk:mb:%s", meta.Prefix, uid)
		// 查询当前未处理的订单
		current, err := meta.MerchantRedis.LLen(ctx, key).Result()
		if err != nil {
			return "0", pushLog(err, helper.RedisErr)
		}

		// 如果当前未处理的订单小于最大接单数量 则派单给改风控人员
		if current < int64(max) {
			return uid, nil
		}
	}

	// 从头循环到尾,没有找到合适风控用户
	return "0", errors.New(helper.RequestBusy)
}

// RisksCloseAuto 风控人员关闭自己接单或是是关闭风控配置的自动派单
func RisksCloseAuto(uid, adminName string) error {

	// 接单员队列
	risksKey := fmt.Sprintf("%s:risk:receive", meta.Prefix)
	// 是否自动接单
	receiveKey := fmt.Sprintf("%s:risk:auto", meta.Prefix)
	if uid == "" || uid == "0" {

		pipe := meta.MerchantRedis.TxPipeline()
		defer pipe.Close()

		//关闭自动接单
		pipe.Unlink(ctx, risksKey)
		pipe.Unlink(ctx, receiveKey)
		_, err := pipe.Exec(ctx)
		if err != nil {
			return pushLog(err, helper.RedisErr)
		}
		contentLog := fmt.Sprintf("风控配置:策略方式-接单控制:后台账号:%s【uid:%s关闭自动接单】", adminName, uid)
		AdminLogInsert(RiskModel, contentLog, CloseOp, adminName)
		return nil
	}

	//如果是关闭单个用户，则删除指定的UID
	_, err := meta.MerchantRedis.LRem(ctx, risksKey, 0, uid).Result()
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}
	userName, _ := AdminGetName(uid)
	contentLog := fmt.Sprintf("风控配置:策略方式-接单控制:后台账号:%s【uid:%s关闭会员：%s】", adminName, uid, userName)
	AdminLogInsert(RiskModel, contentLog, CloseOp, adminName)

	return nil
}

// RisksOpenAuto 开启自动派单或者设置单个风控人员的自动派单
func RisksOpenAuto(uid, adminName string) error {

	// 接单员队列
	risksKey := fmt.Sprintf("%s:risk:receive", meta.Prefix)
	// 是否自动接单
	receiveKey := fmt.Sprintf("%s:risk:auto", meta.Prefix)
	if uid == "" || uid == "0" {

		var ids []string
		ex := g.Ex{
			"state":  WithdrawReviewing,
			"prefix": meta.Prefix,
		}
		t := dialect.From("tbl_withdraw")
		query, _, _ := t.Select("id").Where(ex).ToSQL()
		err := meta.MerchantDB.Select(&ids, query)
		if err != nil {
			return pushLog(err, helper.DBErr)
		}

		_, err = meta.MerchantRedis.Set(ctx, receiveKey, "1", 0).Result()
		if err != nil {
			return pushLog(err, helper.RedisErr)
		}

		/*
			// 所有未派发提款订单加入队列
			for _, v := range ids {
				param := map[string]interface{}{
					"id": v,
				}
				_, _ = BeanPut("risk", param, 0)
			}
		*/
		contentLog := fmt.Sprintf("风控配置-策略方式-接单控制:后台账号:%s【uid:%s开启】", adminName, uid)
		AdminLogInsert(RiskModel, contentLog, SetOp, adminName)
		return nil
	}

	exist, _ := meta.MerchantRedis.Get(ctx, receiveKey).Result()
	if exist != "1" {
		return errors.New(helper.ManualPicking)
	}

	if IsExistRisks(uid) {
		return nil
	}
	//开启指定用户
	_, err := meta.MerchantRedis.LPush(ctx, risksKey, uid).Result()
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}

	userName, _ := AdminGetName(uid)
	contentLog := fmt.Sprintf("风控配置-策略方式-接单控制:后台账号:%s【uid:%s开启%s】", adminName, uid, userName)
	AdminLogInsert(RiskModel, contentLog, SetOp, adminName)
	return nil
}

// SetRisksOrder 删除或者新增list的订单号
func SetRisksOrder(uid, billNo string, diff int) error {

	if uid == "" || uid == "0" || billNo == "" {
		return errors.New(helper.ParamNull)
	}

	key := fmt.Sprintf("%s:risk:mb:%s", meta.Prefix, uid)
	if diff == -1 {
		_, err := meta.MerchantRedis.LRem(ctx, key, 0, billNo).Result()
		if err != nil {
			return pushLog(err, helper.RedisErr)
		}

		return nil
	}

	_, err := meta.MerchantRedis.LPush(ctx, key, billNo).Result()
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}

	return nil
}

func RisksList() ([]string, error) {

	risksKey := fmt.Sprintf("%s:risk:receive", meta.Prefix)
	uidArr, _ := meta.MerchantRedis.LRange(ctx, risksKey, 0, -1).Result()
	return uidArr, nil
}

// 判断用户是否在list中
func IsExistRisks(uid string) bool {

	// 接单员队列
	risksKey := fmt.Sprintf("%s:risk:receive", meta.Prefix)
	// 是否自动接单
	receiveKey := fmt.Sprintf("%s:risk:auto", meta.Prefix)
	if uid == "" || uid == "0" {
		exist, _ := meta.MerchantRedis.Get(ctx, receiveKey).Result()
		if exist == "1" {
			return true
		}
		return false
	}

	total, err := meta.MerchantRedis.LLen(ctx, risksKey).Result()
	if err != nil || total < 1 {
		return false
	}

	uidArr, err := RisksList()
	if err != nil || len(uidArr) < 1 {
		return false
	}
	for _, v := range uidArr {
		if uid == v {
			return true
		}
	}

	return false
}

func SetOrderNum(num, adminName string) error {

	numInt, _ := strconv.Atoi(num)
	if numInt < 1 {
		return errors.New(helper.OrderNumErr)
	}

	maxKey := fmt.Sprintf("%s:risk:maxreceivenum", meta.Prefix)
	_, err := meta.MerchantRedis.Set(ctx, maxKey, numInt, 0).Result()
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}
	contentLog := fmt.Sprintf("风控配置-策略方式-设置接单人数量:后台账号:%s【%s】", adminName, num)
	AdminLogInsert(RiskModel, contentLog, SetOp, adminName)
	return nil
}

func RisksReceives() (ReceiveData, error) {

	var data ReceiveData
	ex := g.Ex{
		"state":    1,
		"group_id": g.Op{"in": []interface{}{"3", "4", "5"}},
		"prefix":   meta.Prefix,
	}
	query, _, _ := dialect.From("tbl_admins").Select("id", "name").Where(ex).Order(g.C("name").Desc()).ToSQL()
	err := meta.MerchantDB.Select(&data.D, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	key := fmt.Sprintf("%s:sms:enablemod", meta.Prefix)
	if meta.MerchantRedis.Exists(ctx, key).Val() > 0 {
		data.EnableMod = true
	}

	return data, nil
}

func RisksNumber() (uint64, error) {

	maxKey := fmt.Sprintf("%s:risk:maxreceivenum", meta.Prefix)
	num, err := meta.MerchantRedis.Get(ctx, maxKey).Uint64()
	if err != nil && err != redis.Nil {
		return num, pushLog(err, helper.RedisErr)
	}

	return num, nil
}

func SetRegMax(num, adminName string) error {

	numInt, _ := strconv.Atoi(num)
	if numInt < 1 {
		return errors.New(helper.OrderNumErr)
	}

	maxKey := fmt.Sprintf("%s:risk:maxregnum", meta.Prefix)
	_, err := meta.MerchantRedis.Set(ctx, maxKey, numInt, 0).Result()
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}

	contentLog := fmt.Sprintf("风控配置-策略方式-注册规则:后台账号:%s【%s】", adminName, num)
	AdminLogInsert(RiskModel, contentLog, SetOp, adminName)
	return nil
}

func RisksRegMax() (uint64, error) {

	maxKey := fmt.Sprintf("%s:risk:maxregnum", meta.Prefix)
	num, err := meta.MerchantRedis.Get(ctx, maxKey).Uint64()
	if err != nil && err != redis.Nil {
		return num, pushLog(err, helper.RedisErr)
	}

	return num, nil
}

// 是否开启关闭提款短信验证
func MemberSmsEnableMod(enable bool, adminName string) error {

	key := fmt.Sprintf("%s:sms:enablemod", meta.Prefix)
	// 开启提款短信验证
	if enable {
		pipe := meta.MerchantRedis.TxPipeline()
		defer pipe.Close()

		pipe.Set(ctx, key, "1", 0)
		pipe.Persist(ctx, key)

		_, err := pipe.Exec(ctx)
		if err != nil {
			return pushLog(err, helper.RedisErr)
		}
	} else { //关闭提款短信验证
		cmd := meta.MerchantRedis.Del(ctx, key)
		fmt.Println(cmd.String())
		err := cmd.Err()
		if err != nil {
			return pushLog(err, helper.RedisErr)
		}
	}

	contentLog := fmt.Sprintf("风控配置-策略方式:后台账号:%s【操作:%s短信验证】", adminName, StateBoolMap[enable])
	AdminLogInsert(RiskModel, contentLog, opBoolMap[enable], adminName)
	return nil
}
