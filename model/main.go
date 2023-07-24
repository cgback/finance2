package model

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"finance/contrib/tracerr"
	ryrpc "finance/rpc"
	"fmt"
	"github.com/lucacasonato/mqtt"
	"github.com/shopspring/decimal"
	"math/rand"
	"strconv"

	"github.com/apache/rocketmq-client-go/v2"
	g "github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/qiniu/qmgo"
	rycli "github.com/ryrpc/client"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"
	"runtime"
	"strings"
	"time"
)

type MetaTable struct {
	MerchantDB     *sqlx.DB
	MerchantTD     *sqlx.DB
	MerchantRedis  *redis.Client
	MerchantMQ     rocketmq.Producer
	MgCli          *qmgo.Client
	MgDB           *qmgo.Database
	MerchantMqtt   *mqtt.Client
	MerchantRPC    *rycli.Client
	Program        string
	Prefix         string
	PayRPC         string
	Finance        map[string]map[string]interface{}
	FcallbackInner string
	IsDirect       int
}

var (
	loc                     *time.Location
	meta                    *MetaTable
	ctx                     = context.Background()
	dialect                 = g.Dialect("mysql")
	fc                      *fasthttp.Client
	vnPay                   Payment
	zero                    = decimal.NewFromInt(0)
	colCate                 = helper.EnumFields(Category{})
	colsChannelType         = helper.EnumFields(ChannelType{})
	colPayment              = helper.EnumFields(Payment_t{})
	colsBankCard            = helper.EnumFields(Bankcard_t{})
	coleBankTypes           = helper.EnumFields(TblBankTypes{})
	colsVirtualWallet       = helper.EnumFields(VirtualWallet_t{})
	colsPayment             = helper.EnumFields(Payment_t{})
	colsMemberVirtualWallet = helper.EnumFields(MemberVirtualWallet{})
	colsMember              = helper.EnumFields(Member{})
	colsMemberInfo          = helper.EnumFields(MemberInfo{})
	colsDeposit             = helper.EnumFields(Deposit{})
	colsWithdraw            = helper.EnumFields(Withdraw{})
	colsMemberBankcard      = helper.EnumFields(MemberBankCard{})
	colsConfig              = helper.EnumFields(FConfig{})
	colsMemberConfig        = helper.EnumFields(FMemberConfig{})
	colsMemberLock          = helper.EnumFields(MemberLock{})
)

func Constructor(mt *MetaTable, payRPC string) {

	meta = mt
	loc, _ = time.LoadLocation("Asia/Bangkok")
	ryrpc.Constructor(payRPC)

	err := Lock(meta.Prefix + "_f2_load")
	if err == nil {
		LoadChannelType()
		CacheRefreshLevel()
		CateListRedis()
		BankCardUpdateCache()
	}

	fc = &fasthttp.Client{
		MaxConnsPerHost: 60000,
		TLSConfig:       &tls.Config{InsecureSkipVerify: true},
		ReadTimeout:     time.Second * 10,
		WriteTimeout:    time.Second * 10,
	}
	NewPayment()
}

func pushLog(err error, code string) error {

	_, file, line, _ := runtime.Caller(1)
	paths := strings.Split(file, "/")
	l := len(paths)
	if l > 2 {
		file = paths[l-2] + "/" + paths[l-1]
	}
	path := fmt.Sprintf("%s:%d", file, line)

	id := helper.GenId()
	ts := time.Now()
	data := map[string]string{
		"id":       id,
		"content":  tracerr.SprintSource(err, 2, 2),
		"flags":    code,
		"filename": path,
		"_index":   fmt.Sprintf("%s_%s_%04d%02d", meta.Prefix, meta.Program, ts.Year(), ts.Month()),
	}
	payload, _ := helper.JsonMarshal(data)
	fmt.Println(string(payload))
	_ = RocketSendAsync("zinc_fluent_log", payload)

	return fmt.Errorf("hệ thống lỗi %s", id)
}

func Close() {
}

func AdminToken(ctx *fasthttp.RequestCtx) (map[string]string, error) {

	b := ctx.UserValue("token").([]byte)

	var p fastjson.Parser

	data := map[string]string{}
	v, err := p.ParseBytes(b)
	if err != nil {
		return data, err
	}

	o, err := v.Object()
	if err != nil {
		return data, err
	}

	o.Visit(func(k []byte, v *fastjson.Value) {
		key := string(k)
		val, err := v.StringBytes()
		if err == nil {
			data[key] = string(val)
		}
	})

	return data, nil
}

func pushWithdrawNotify(format, username, amount string) error {

	ts := time.Now()
	msg := fmt.Sprintf(format, username, amount, username, amount, username, amount)
	msg = strings.TrimSpace(msg)

	topic := fmt.Sprintf("%s/merchant", meta.Prefix)
	err := meta.MerchantMqtt.Publish(ctx, topic, []byte(msg), mqtt.AtLeastOnce)
	if err != nil {
		fmt.Println("failed", time.Since(ts), err.Error())
		fmt.Println("merchantNats.Publish finance = ", err.Error())
		return err
	}

	return nil
}

func randSliceValue(xs []int) int {

	rand.Seed(time.Now().Unix())
	return xs[rand.Intn(len(xs))]
}

// 获取admin的name
func AdminGetName(id string) (string, error) {

	var name string
	query, _, _ := dialect.From("tbl_admins").Select("name").Where(g.Ex{"id": id}).ToSQL()
	err := meta.MerchantDB.Get(&name, query)
	if err != nil && err != sql.ErrNoRows {
		return name, pushLog(err, helper.DBErr)
	}

	return name, nil
}

// TimeFormat 时间戳转化成string
func TimeFormat(timestamp int64) string {

	t := time.Unix(timestamp, 0).In(loc)
	return t.Format("2006-01-02 15:04:05")
}

func tdInsert(tbl string, record g.Record) {

	query, _, _ := dialect.Insert(tbl).Rows(record).ToSQL()
	fmt.Println(query)
	_, err := meta.MerchantTD.Exec(query)
	if err != nil {
		fmt.Println("update td = ", err.Error(), record)
	}
}

func CheckSmsCaptcha(ip, ts, sid, phone, code string) (bool, error) {

	key := fmt.Sprintf("%s:sms:%s%s", meta.Prefix, phone, sid)
	cmd := meta.MerchantRedis.Get(ctx, key)
	val, err := cmd.Result()
	if err != nil && err != redis.Nil {
		return false, pushLog(fmt.Errorf("CheckSmsCaptcha cmd : %s ,error : %s ", cmd.String(), err.Error()), helper.RedisErr)
	}

	if code == val {
		its, _ := strconv.ParseInt(ts, 10, 64)
		tdInsert("sms_log", g.Record{
			"ts":         its,
			"state":      "2",
			"updated_at": time.Now().Unix(),
		})
		return true, nil
	}

	return false, errors.New(helper.CaptchaErr)
}

// WithdrawFind 查找单条提款记录, 订单不存在返回错误: OrderNotExist
func WithdrawFind(id string) (Withdraw, error) {

	w := Withdraw{}
	query, _, _ := dialect.From("tbl_withdraw").Select(colsWithdraw...).Where(g.Ex{"id": id}).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&w, query)
	if err == sql.ErrNoRows {
		return w, errors.New(helper.OrderNotExist)
	}

	if err != nil {
		return w, pushLog(err, helper.DBErr)
	}

	return w, nil
}

func WithdrawGetBank(bid, username string) (MemberBankCard, error) {

	bank := MemberBankCard{}
	banks, err := MemberBankcardList(g.Ex{"username": username})
	if err != nil && err != sql.ErrNoRows {
		return bank, err
	}

	for _, v := range banks {
		if v.ID == bid {
			bank = v
			break
		}
	}
	if bank.ID != bid {
		return bank, errors.New(helper.BankcardIDErr)
	}

	return bank, nil
}

// 获取银行卡成功失败的次数
func WithdrawBanKCardNumber(bid string) (int, int) {

	ex := g.Ex{
		"bid":    bid,
		"prefix": meta.Prefix,
	}
	var data []StateNum
	var successNum int
	var failNum int
	query, _, _ := dialect.From("tbl_withdraw").Select(g.COUNT("id").As("t"), g.C("state").As("state")).Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&data, query)
	if err != nil && err != sql.ErrNoRows {
		return 0, 0
	}
	for _, v := range data {
		if v.State == WithdrawSuccess {
			successNum += v.T
		}
		if v.State == WithdrawReviewReject || v.State == WithdrawAbnormal || v.State == WithdrawFailed {
			failNum += v.T
		}
	}

	return successNum, failNum
}

// 金额对比
func compareAmount(compare, compared string, cent int64) error {

	ca, err := decimal.NewFromString(compare)
	if err != nil {
		return errors.New("parse amount error")
	}

	ra, err := decimal.NewFromString(compared)
	if err != nil {
		return errors.New("parse amount error")
	}

	// 数据库的金额是k为单位 ra.Mul(decimal.NewFromInt(1000))
	if ca.Cmp(ra.Mul(decimal.NewFromInt(cent))) != 0 {
		return errors.New("invalid amount")
	}

	return nil
}
