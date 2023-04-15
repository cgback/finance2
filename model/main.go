package model

import (
	"context"
	"finance/contrib/helper"
	"finance/contrib/tracerr"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	g "github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"runtime"
	"strings"
	"time"
)

type MetaTable struct {
	MerchantTD   *sqlx.DB
	MerchantPika *redis.Client
	MerchantMQ   rocketmq.Producer
	Program      string
	Prefix       string
}

var (
	loc     *time.Location
	meta    *MetaTable
	ctx     = context.Background()
	dialect = g.Dialect("mysql")
)

func Constructor(mt *MetaTable) {

	meta = mt
	loc, _ = time.LoadLocation("Asia/Shanghai")
}

func pushLog(err error, code string) error {

	fmt.Println(err)
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
	err = meta.MerchantMQ.SendAsync(ctx,
		func(c context.Context, result *primitive.SendResult, e error) {
			if e != nil {
				fmt.Printf("send message error: %s\n", e.Error())
			}
		}, primitive.NewMessage("zinc_fluent_log", payload))
	if err != nil {
		fmt.Printf("rocket SendAsync payload[%s] error[%s]", string(payload), err.Error())
	}

	return fmt.Errorf("系统错误 %s", id)
}

func Close() {
}
