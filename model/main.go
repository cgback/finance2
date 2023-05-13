package model

import (
	"context"
	"finance/contrib/helper"
	"finance/contrib/tracerr"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2"
	g "github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"runtime"
	"strings"
	"time"
)

type MetaTable struct {
	MerchantDB   *sqlx.DB
	MerchantPika *redis.Client
	MerchantMQ   rocketmq.Producer
	Program      string
	Prefix       string
}

var (
	loc             *time.Location
	meta            *MetaTable
	ctx             = context.Background()
	dialect         = g.Dialect("mysql")
	colsChannelType = helper.EnumFields(ChannelType{})
)

func Constructor(mt *MetaTable) {

	meta = mt
	loc, _ = time.LoadLocation("Asia/Shanghai")
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
