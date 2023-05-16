package middleware

import (
	"errors"
	"finance/contrib/session"
	"fmt"
	"github.com/valyala/fasthttp"
)

var allows = map[string]bool{
	"/merchant/report/version":            true,
	"/merchant/report/pprof/":             true,
	"/merchant/report/pprof/block":        true,
	"/merchant/report/pprof/allocs":       true,
	"/merchant/report/pprof/cmdline":      true,
	"/merchant/report/pprof/goroutine":    true,
	"/merchant/report/pprof/heap":         true,
	"/merchant/report/pprof/profile":      true,
	"/merchant/report/pprof/trace":        true,
	"/merchant/report/pprof/threadcreate": true,
}

func CheckTokenMiddleware(ctx *fasthttp.RequestCtx) error {

	path := string(ctx.Path())
	fmt.Println(path)
	if _, ok := allows[path]; ok {
		return nil
	}

	data, err := session.Get(ctx)
	if err != nil {
		return errors.New(`{"status":false,"data":"token"}`)
	}

	ctx.SetUserValue("token", data)

	return nil
}
