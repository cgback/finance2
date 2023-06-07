package router

import (
	"fmt"
	"runtime/debug"
	"time"

	"finance/controller"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

var (
	ApiTimeoutMsg = `{"status": "false","data":"服务器响应超时，请稍后重试"}`
	ApiTimeout    = time.Second * 30
	route         *router.Router
	buildInfo     BuildInfo
)

type BuildInfo struct {
	GitReversion   string
	BuildTime      string
	BuildGoVersion string
}

func apiServerPanic(ctx *fasthttp.RequestCtx, rcv interface{}) {

	err := rcv.(error)
	fmt.Println(err)
	debug.PrintStack()

	if r := recover(); r != nil {
		fmt.Println("recovered failed", r)
	}

	ctx.SetStatusCode(500)
	return
}

func Version(ctx *fasthttp.RequestCtx) {

	ctx.SetContentType("text/html; charset=utf-8")
	fmt.Fprintf(ctx, "reportApi<br />Git reversion = %s<br />Build Time = %s<br />Go version = %s<br />System Time = %s<br />",
		buildInfo.GitReversion, buildInfo.BuildTime, buildInfo.BuildGoVersion, ctx.Time())

	//ctx.Request.Header.VisitAll(func (key, value []byte) {
	//	fmt.Fprintf(ctx, "%s: %s<br/>", string(key), string(value))
	//})
}

// SetupRouter 设置路由列表
func SetupRouter(b BuildInfo) *router.Router {

	route = router.New()
	route.PanicHandler = apiServerPanic

	buildInfo = b

	// 日志服务
	channelTypeCtl := new(controller.ChannelTypeController)
	// 收款账号管理
	bankCardCtl := new(controller.BankCardController)
	usdtCtl := new(controller.UsdtController)

	get("/f2/version", Version)
	// 渠道管理-列表
	get("/merchant/f2/channel/type/list", channelTypeCtl.List)
	// 渠道管理-列表-更新状态
	get("/merchant/f2/channel/type/updatestate", channelTypeCtl.UpdateState)
	// 渠道管理-列表-修改排序
	get("/merchant/f2/channel/type/updatesort", channelTypeCtl.UpdateSort)
	// 渠道管理-收款账户管理
	get("/merchant/f2/offline/bankcard/list", bankCardCtl.List)
	// 渠道管理-收款账户管理-添加银行卡
	post("/merchant/f2/offline/bankcard/insert", bankCardCtl.Insert)
	// 渠道管理-收款账户管理-更新银行卡
	post("/merchant/f2/offline/bankcard/update", bankCardCtl.Update)
	// 渠道管理-收款账户管理-删除银行卡
	get("/merchant/f2/offline/bankcard/delete", bankCardCtl.Delete)
	// 渠道管理-收款账户管理-usdt汇率展示
	get("/merchant/f2/offline/usdt/info", usdtCtl.Info)
	// 渠道管理-收款账户管理-usdt汇率修改配置
	post("/merchant/f2/offline/usdt/update", usdtCtl.Update)
	// 渠道管理-收款账户管理-usdt设置-添加usdt收款账号
	post("/merchant/f2/offline/usdt/insert", usdtCtl.Insert)
	// 渠道管理-收款账户管理-usdt设置-展示usdt收款账号
	post("/merchant/f2/offline/usdt/list", usdtCtl.List)
	// 渠道管理-收款账户管理-usdt设置-展示usdt收款账号
	post("/merchant/f2/offline/usdt/update/account", usdtCtl.UpdateAccount)
	// 渠道管理-收款账户管理-usdt设置-展示usdt收款账号
	get("/merchant/f2/offline/usdt/delete", usdtCtl.Delete)
	// 渠道管理-收款账户管理-usdt设置-展示usdt收款账号
	post("/merchant/f2/offline/usdt/updatestate", usdtCtl.UpdateState)
	//
	return route
}

// get is a shortcut for router.GET(path string, handle fasthttp.RequestHandler)
func get(path string, handle fasthttp.RequestHandler) {
	route.GET(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
}

// head is a shortcut for router.HEAD(path string, handle fasthttp.RequestHandler)
func head(path string, handle fasthttp.RequestHandler) {
	route.HEAD(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
}

// options is a shortcut for router.OPTIONS(path string, handle fasthttp.RequestHandler)
func options(path string, handle fasthttp.RequestHandler) {
	route.OPTIONS(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
}

// post is a shortcut for router.POST(path string, handle fasthttp.RequestHandler)
func post(path string, handle fasthttp.RequestHandler) {
	route.POST(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
}

// put is a shortcut for router.PUT(path string, handle fasthttp.RequestHandler)
func put(path string, handle fasthttp.RequestHandler) {
	route.PUT(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
}

// patch is a shortcut for router.PATCH(path string, handle fasthttp.RequestHandler)
func patch(path string, handle fasthttp.RequestHandler) {
	route.PATCH(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
}

// delete is a shortcut for router.DELETE(path string, handle fasthttp.RequestHandler)
func delete(path string, handle fasthttp.RequestHandler) {
	route.DELETE(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
}
