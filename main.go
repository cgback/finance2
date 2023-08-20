package main

import (
	"context"
	"finance/contrib/apollo"
	"finance/contrib/conn"
	"finance/contrib/helper"
	"finance/contrib/session"
	"finance/middleware"
	"finance/model"
	"finance/router"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/lucacasonato/mqtt"
	rycli "github.com/ryrpc/client"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/producer"

	"github.com/valyala/fasthttp"
	_ "go.uber.org/automaxprocs"
)

func main() {

	var ctx = context.Background()
	argc := len(os.Args)
	if argc != 4 {
		fmt.Printf("%s <etcds> <cfgPath>\r\n", os.Args[0])
		return
	}

	cfg := conf{}
	endpoints := strings.Split(os.Args[1], ",")
	apollo.New(endpoints, ETCDName, ETCDPass)
	err := apollo.ParseTomlStruct(os.Args[2], &cfg)
	content, err := apollo.ParseToml(path.Dir(os.Args[2])+"/finance.toml", false)
	apollo.Close()
	if err != nil {
		fmt.Printf("ParseTomlStruct error: %s", err.Error())
		return
	}

	mt := new(model.MetaTable)
	mt.Prefix = cfg.Prefix
	conn.Use(validateKey)
	mt.MerchantDB = conn.InitDB(cfg.Db.Masteren.Addr, cfg.Db.Masteren.MaxIdleConn, cfg.Db.Masteren.MaxOpenConn)
	mt.MerchantRedis = conn.InitRedisSentinel(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.Sentinel, 0)
	mt.MgCli, mt.MgDB = conn.InitMongo(ctx, cfg.Mongodb.Url, cfg.Mongodb.Username, cfg.Mongodb.Password, cfg.Mongodb.Db)
	mt.MerchantTD = conn.InitTD(cfg.Td.Message.Addr, cfg.Td.Message.MaxIdleConn, cfg.Td.Message.MaxOpenConn)
	mt.FcallbackInner = cfg.Fcallback

	mt.MerchantMQ, err = rocketmq.NewProducer(
		producer.WithNameServer(cfg.Rocketmq),
		producer.WithRetry(2),
		producer.WithGroupName("finance2"),
	)
	if err != nil {
		fmt.Printf("start NewProducer error: %s", err.Error())
		os.Exit(1)
	}
	err = mt.MerchantMQ.Start()
	if err != nil {
		fmt.Printf("start producer error: %s", err.Error())
		os.Exit(1)
	}
	mt.MerchantMqtt, err = mqtt.NewClient(mqtt.ClientOptions{
		// required
		Servers: cfg.Nats.Servers,

		// optional
		ClientID:      helper.GenId(),
		Username:      cfg.Nats.Username,
		Password:      cfg.Nats.Password,
		AutoReconnect: true,
	})
	if err != nil {
		panic(err)
	}

	err = mt.MerchantMqtt.Connect(ctx)
	if err != nil {
		panic(err)
	}

	mt.MerchantRPC = rycli.NewClient()
	mt.Finance = content

	model.Constructor(mt, cfg.RPC)

	defer func() {
		model.Close()
		mt = nil
	}()

	if os.Args[3] == "load" {
		fmt.Println("load")
		model.BankTypeUpdateCache()
		model.BankCardUpdateCache()
	}

	mt.Program = filepath.Base(os.Args[0])
	session.New(mt.MerchantRedis, mt.Prefix)
	b := router.BuildInfo{
		GitReversion:   gitReversion,
		BuildTime:      buildTime,
		BuildGoVersion: buildGoVersion,
	}
	app := router.SetupRouter(b)
	srv := &fasthttp.Server{
		Handler:            middleware.Use(app.Handler, mt.Prefix, validateH5, validateHT, validateWEB, validateAndroid, validateIOS, true),
		ReadTimeout:        router.ApiTimeout,
		WriteTimeout:       router.ApiTimeout,
		Name:               "finance2",
		MaxRequestBodySize: 51 * 1024 * 1024,
	}
	fmt.Printf("gitReversion = %s\r\nbuildGoVersion = %s\r\nbuildTime = %s\r\n", gitReversion, buildGoVersion, buildTime)
	fmt.Println("finance2 running", cfg.Port.Finance2)
	// 启动小飞机推送版本信息
	if !cfg.IsDev {
		model.TelegramBotNotice(mt.Program, username, gitReversion, buildTime, buildGoVersion, "rpc", cfg.Prefix, cfg.Sock5, cfg.Env, cfg.Tg.BotID, cfg.Tg.NoticeGroupID)
	}

	helper.Use(validateH5, validateHT, validateWEB, validateAndroid, validateIOS, true)
	if err := srv.ListenAndServe(cfg.Port.Finance2); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}
