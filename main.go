package main

import (
	"finance/contrib/apollo"
	"finance/contrib/conn"
	"finance/middleware"
	"finance/model"
	"finance/router"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"log"
	"os"
	"strings"

	"github.com/valyala/fasthttp"
	_ "go.uber.org/automaxprocs"
)

var (
	gitReversion   = ""
	buildTime      = ""
	buildGoVersion = ""
)

func main() {

	argc := len(os.Args)
	if argc != 3 {
		fmt.Printf("%s <etcds> <cfgPath>\r\n", os.Args[0])
		return
	}

	cfg := conf{}
	endpoints := strings.Split(os.Args[1], ",")
	apollo.New(endpoints)
	err := apollo.ParseTomlStruct(os.Args[2], &cfg)
	apollo.Close()
	if err != nil {
		fmt.Printf("ParseTomlStruct error: %s", err.Error())
		return
	}

	mt := new(model.MetaTable)
	mt.MerchantTD = conn.InitDB(cfg.Maindb.Addr, cfg.Maindb.MaxIdleConn, cfg.Maindb.MaxOpenConn)
	mt.MerchantPika = conn.InitRedisSentinel(cfg.Pika.Addr, cfg.Pika.Password, cfg.Pika.Sentinel, 0)
	mt.MerchantMQ, err = rocketmq.NewProducer(
		producer.WithNameServer(cfg.Rocketmq),
		producer.WithRetry(2),
		producer.WithGroupName("merchant"),
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

	bin := strings.Split(os.Args[0], "/")
	mt.Program = bin[len(bin)-1]
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
		Name:               "lotteryinfo",
		MaxRequestBodySize: 51 * 1024 * 1024,
	}
	fmt.Printf("gitReversion = %s\r\nbuildGoVersion = %s\r\nbuildTime = %s\r\n", gitReversion, buildGoVersion, buildTime)
	fmt.Println("lotteryinfo running", cfg.Port)
	if err := srv.ListenAndServe(cfg.Port); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}
