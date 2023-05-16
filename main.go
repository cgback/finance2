package main

import (
	"finance/contrib/apollo"
	"finance/contrib/conn"
	"finance/contrib/session"
	"finance/middleware"
	"finance/model"
	"finance/router"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/producer"

	"github.com/valyala/fasthttp"
	_ "go.uber.org/automaxprocs"
)

func main() {

	argc := len(os.Args)
	if argc != 4 {
		fmt.Printf("%s <etcds> <cfgPath>\r\n", os.Args[0])
		return
	}

	cfg := conf{}
	endpoints := strings.Split(os.Args[1], ",")
	apollo.New(endpoints, ETCDName, ETCDPass)
	err := apollo.ParseTomlStruct(os.Args[2], &cfg)
	apollo.Close()
	if err != nil {
		fmt.Printf("ParseTomlStruct error: %s", err.Error())
		return
	}

	mt := new(model.MetaTable)
	conn.Use(validateKey)
	mt.MerchantDB = conn.InitDB(cfg.Db.Masteren.Addr, cfg.Db.Masteren.MaxIdleConn, cfg.Db.Masteren.MaxOpenConn)
	mt.MerchantRedis = conn.InitRedisSentinel(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.Sentinel, 0)
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

	model.Constructor(mt)

	defer func() {
		model.Close()
		mt = nil
	}()

	mt.Program = filepath.Base(os.Args[0])
	session.New(mt.MerchantRedis, mt.Prefix)
	b := router.BuildInfo{
		GitReversion:   gitReversion,
		BuildTime:      buildTime,
		BuildGoVersion: buildGoVersion,
	}
	app := router.SetupRouter(b)
	srv := &fasthttp.Server{
		Handler:            middleware.Use(app.Handler),
		ReadTimeout:        router.ApiTimeout,
		WriteTimeout:       router.ApiTimeout,
		Name:               "finance2",
		MaxRequestBodySize: 51 * 1024 * 1024,
	}
	fmt.Printf("gitReversion = %s\r\nbuildGoVersion = %s\r\nbuildTime = %s\r\n", gitReversion, buildGoVersion, buildTime)
	fmt.Println("finance2 running", cfg.Port.Finance2)
	if err := srv.ListenAndServe(cfg.Port.Finance2); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}
