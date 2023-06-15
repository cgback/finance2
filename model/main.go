package model

import (
	"context"
	"database/sql"
	"finance/contrib/helper"
	"finance/contrib/tracerr"
	"fmt"
	"github.com/lucacasonato/mqtt"
	"math/rand"

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
	MerchantDB    *sqlx.DB
	MerchantRedis *redis.Client
	MerchantMQ    rocketmq.Producer
	MgCli         *qmgo.Client
	MgDB          *qmgo.Database
	MerchantMqtt  *mqtt.Client
	MerchantRPC   *rycli.Client
	Program       string
	Prefix        string
	PayRPC        string
}

var (
	loc                     *time.Location
	meta                    *MetaTable
	ctx                     = context.Background()
	dialect                 = g.Dialect("mysql")
	fc                      *fasthttp.Client
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
)

func Constructor(mt *MetaTable, payRPC string) {

	meta = mt
	loc, _ = time.LoadLocation("Asia/Bangkok")

	meta.MerchantRPC.SetBaseURL(payRPC)
	meta.MerchantRPC.SetClientTimeout(12 * time.Second)

	err := Lock(meta.Prefix + "_finance2_load")
	if err == nil {
		LoadChannelType()
	}

	data, err := rpcDepositChannelList("91408276484425095")
	if err != nil {
		_ = pushLog(err, helper.GetRPCErr)
	}

	fmt.Printf("%#v \r\n", data)
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
