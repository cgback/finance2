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

	//充值
	payCtl := new(controller.PayController)
	// 支付方式
	channelTypeCtl := new(controller.ChannelTypeController)
	// 收款账号管理
	bankCardCtl := new(controller.BankCardController)
	// 收款usdt账号管理
	usdtCtl := new(controller.UsdtController)
	//风控管理
	risksCtl := new(controller.RisksController)
	//通道管理
	paymentCtl := new(controller.PaymentController)
	//渠道管理
	cateCtl := new(controller.CateController)
	//线下充值
	manualCtl := new(controller.ManualController)
	//存款管理
	depositCtl := new(controller.DepositController)
	//提款管理
	wdCtl := new(controller.WithdrawController)
	cbCtl := new(controller.CallBackController)

	get("/f2/version", Version)
	// [callback] NVN 代付回调
	post("/f2/callback/nvnw", cbCtl.NVNW)
	// [callback] NVN 代收回调
	post("/f2/callback/nvnd", cbCtl.NVND)
	// 前台充值方式
	get("/f2/cate", payCtl.Cate)
	// 前台充值通道
	get("/f2/tunnel", payCtl.Tunnel)
	// 前台充值
	post("/f2/pay", payCtl.Pay)
	// [前台] 线下转卡-发起存款
	post("/f2/gen/code", manualCtl.GenCode)

	// [前台] 用户申请提现
	post("/f2/withdraw", wdCtl.Withdraw)
	// [前台] 用户提现剩余次数和额度
	get("/f2/withdraw/limit", wdCtl.Limit)
	// [前台] 获取正在处理中的提现订单
	get("/f2/withdraw/processing", wdCtl.Processing)
	// [前台] 渠道列表数据缓存
	get("/f2/cate/cache", cateCtl.Cache)

	//财务管理-渠道列表
	post("/merchant/f2/cate/list", cateCtl.List)
	//财务管理-渠道状态修改
	post("/merchant/f2/cate/update/state", cateCtl.UpdateState)

	//  财务管理-渠道管理-通道管理-修改
	post("/merchant/f2/payment/update", paymentCtl.Update)
	// 财务管理-渠道管理-通道管理-列表
	get("/merchant/f2/payment/list", paymentCtl.List)
	//  财务管理-渠道管理-通道管理-启用/停用
	post("/merchant/f2/payment/update/state", paymentCtl.UpdateState)

	// 渠道管理-支付方式列表
	get("/merchant/f2/channel/type/list", channelTypeCtl.List)
	// 渠道管理-支付方式列表-更新状态
	get("/merchant/f2/channel/type/updatestate", channelTypeCtl.UpdateState)
	// 渠道管理-支付方式列表-修改排序
	get("/merchant/f2/channel/type/updatesort", channelTypeCtl.UpdateSort)

	// 渠道管理-收款账户管理
	get("/merchant/f2/offline/bankcard/list", bankCardCtl.List)
	// 渠道管理-收款账户管理-添加银行卡
	post("/merchant/f2/offline/bankcard/insert", bankCardCtl.Insert)
	// 渠道管理-收款账户管理-更新银行卡
	post("/merchant/f2/offline/bankcard/update", bankCardCtl.Update)
	// 渠道管理-收款账户管理-删除银行卡
	get("/merchant/f2/offline/bankcard/delete", bankCardCtl.Delete)
	// 渠道管理-收款账户管理-添加银行卡充值说明
	post("/merchant/f2/offline/bankcard/msg/insert", bankCardCtl.InsertMsg)

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

	// [商户后台] 风控管理-风控配置-接单控制-关闭自动派单
	get("/merchant/f2/risks/close", risksCtl.CloseAuto)
	// [商户后台] 风控管理-风控配置-接单控制-开启自动派单
	get("/merchant/f2/risks/open", risksCtl.OpenAuto)
	// [商户后台] 风控管理-风控配置-获取自动派单状态
	get("/merchant/f2/risks/state", risksCtl.State)
	// [商户后台] 风控管理-风控配置-获取自动派单人员的列表
	get("/merchant/f2/risks/list", risksCtl.List)
	// [商户后台] 风控管理-风控配置-设置接单数量
	get("/merchant/f2/risks/setnumer", risksCtl.SetNumber)
	// [商户后台] 风控管理-风控配置-领取人列表
	get("/merchant/f2/risks/receives", risksCtl.Receives)
	// [商户后台] 风控管理-风控配置-领取人数量
	get("/merchant/f2/risks/number", risksCtl.Number)
	// [商户后台] 风控管理-风控配置-设置同设备号注册数量
	post("/merchant/f2/risks/setregmax", risksCtl.SetRegMax)
	// [商户后台] 风控管理-风控配置-获取同设备号注册数量
	get("/merchant/f2/risks/regmax", risksCtl.RegMax)
	// [商户后台] 风控管理-风控配置-每日提现验证码开启关闭控制按钮
	get("/merchant/f2/risks/check/daily", risksCtl.EnableMod)

	// [商户后台] 会员列表-存款管理-存款信息
	get("/merchant/f2/deposit/detail", depositCtl.Detail)
	// [商户后台] 财务管理-存款管理-入款订单列表/补单审核列表
	get("/merchant/f2/deposit/list", depositCtl.List)
	// [商户后台] 财务管理-存款管理-历史记录
	get("/merchant/f2/deposit/history", depositCtl.History)
	// [商户后台] 财务管理-存款管理-存款补单
	post("/merchant/f2/deposit/manual", depositCtl.Manual)
	// [商户后台] 财务管理-存款管理-补单审核
	post("/merchant/f2/deposit/review", depositCtl.Review)
	// [商户后台] 财务管理-存款管理-USDT存款
	post("/merchant/f2/deposit/usdt/list", depositCtl.USDTList)
	// [商户后台] 财务管理-存款管理-线下转卡-入款订单
	post("/merchant/f2/manual/list", manualCtl.List)

	// [商户后台] 财务管理-提款管理-会员列表-提款
	post("/merchant/f2/withdraw/memberlist", wdCtl.MemberWithdrawList)
	// [商户后台] 财务管理-提款管理-提款列表
	post("/merchant/f2/withdraw/financelist", wdCtl.FinanceReviewList)
	// [商户后台] 财务管理-提款管理-提款历史记录
	post("/merchant/f2/withdraw/historylist", wdCtl.HistoryList)
	// [商户后台] 财务管理-提款管理-拒绝
	post("/merchant/f2/withdraw/reject", wdCtl.ReviewReject)
	// [商户后台] 财务管理-提款管理-人工出款（手动代付， 手动出款）
	post("/merchant/f2/withdraw/review", wdCtl.Review)
	// [商户后台] 财务管理-提款管理-代付失败
	post("/merchant/f2/withdraw/automatic/failed", wdCtl.AutomaticFailed)

	// [商户后台] 风控管理-提款审核-待领取列表
	post("/merchant/f2/withdraw/waitreceive", wdCtl.RiskWaitConfirmList)
	// [商户后台] 风控管理-提款审核-待领取列表-银行卡交易记录统计
	post("/merchant/f2/withdraw/cardrecord", wdCtl.BankCardWithdrawRecord)
	// [商户后台] 风控管理-提款审核-待审核列表
	post("/merchant/f2/withdraw/waitreview", wdCtl.RiskReviewList)
	// [商户后台] 风控管理-提款审核-待审核列表-通过
	post("/merchant/f2/withdraw/reviewpass", wdCtl.RiskReview)
	// [商户后台] 风控管理-提款审核-待审核列表-拒绝
	post("/merchant/f2/withdraw/reviewreject", wdCtl.RiskReviewReject)
	// [商户后台] 风控管理-提款审核-待审核列表-挂起
	post("/merchant/f2/withdraw/hangup", wdCtl.HangUp)
	// [商户后台] 风控管理-提款审核-挂起列表
	post("/merchant/f2/withdraw/hanguplist", wdCtl.HangUpList)
	// [商户后台] 风控管理-提款审核-待审核列表-修改领取人
	post("/merchant/f2/withdraw/receiveupdate", wdCtl.ConfirmNameUpdate)
	// [商户后台] 风控管理-提款审核-挂起列表-领取
	post("/merchant/f2/withdraw/receive", wdCtl.ConfirmName)
	// [商户后台] 风控管理-提款审核-历史记录列表
	post("/merchant/f2/withdraw/riskhistory", wdCtl.RiskHistory)

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
