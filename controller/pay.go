package controller

import (
	"finance/contrib/helper"
	"finance/model"

	"github.com/valyala/fasthttp"
)

type PayController struct{}

var onlinePay = map[string]bool{
	"59000000000000001": true, //	QR Banking
	"59000000000000002": true, //	MomoPay
	"59000000000000003": true, //	ZaloPay
	"59000000000000004": true, //	ViettelPay
	"59000000000000005": true, //	TheCao
	"59000000000000101": true, //	bankPay
	//"133221087319615487": true, //	withdraw

}

var offlinePay = map[string]bool{
	"766870294997073616": true, //	offline
	"766870294997073617": true, //	momo转卡
	"766870294997073618": true, //	momo转帐
	"766870294997073619": true, //	VietteIPay转卡
	"766870294997073620": true, //	VietteIPay转帐
	"766870294997073621": true, //	ZaloPay转卡
	//"779402438062874465": true, //	线下usdt提现
	"779402438062874469": true, //	线下usdt
}

func (that *PayController) Pay(ctx *fasthttp.RequestCtx) {

	id := string(ctx.PostArgs().Peek("id"))
	amount := string(ctx.PostArgs().Peek("amount"))
	bid := string(ctx.PostArgs().Peek("bid"))

	if !helper.CtypeDigit(amount) {
		helper.Print(ctx, false, helper.AmountErr)
		return
	}

	// 在线支付走if里面的代码
	if _, ok := onlinePay[id]; ok {
		//fmt.Println("Pay newestPay id = ", id)
		res, err := model.PayOnline(ctx, id, amount, bid)
		if err != nil {
			helper.Print(ctx, false, err.Error())
			return
		}
		helper.Print(ctx, true, res)
		return
	}

	// offline支付走if里面的代码
	if _, ok := offlinePay[id]; ok {
		res, err := model.OfflinePay(ctx, id, amount, bid)
		if err != nil {
			helper.Print(ctx, false, err.Error())
			return
		}
		helper.PrintJson(ctx, true, res)
		return
	}

	helper.PrintJson(ctx, false, "404")
}

func (that *PayController) Tunnel(ctx *fasthttp.RequestCtx) {

	id := string(ctx.QueryArgs().Peek("id"))
	if !helper.CtypeDigit(id) {
		helper.Print(ctx, false, helper.IDErr)
		return
	}

	data, err := model.Tunnel(ctx, id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.PrintJson(ctx, true, data)
}

func (that *PayController) Cate(ctx *fasthttp.RequestCtx) {

	data, err := model.Cate(ctx)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}
