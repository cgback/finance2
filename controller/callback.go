package controller

import (
	"finance/model"
	"github.com/valyala/fasthttp"
)

type CallBackController struct{}

func (that *CallBackController) NVNW(ctx *fasthttp.RequestCtx) {

	//model.WithdrawalCallBack(ctx, model.WPay)
	model.WithdrawalCallBack(ctx)
}

func (that *CallBackController) NVND(ctx *fasthttp.RequestCtx) {

	//model.DepositCallBack(ctx, model.WPay)
	model.DepositCallBack(ctx)
}
