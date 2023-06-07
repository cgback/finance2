package controller

import (
	"finance/contrib/helper"
	"finance/contrib/validator"
	"finance/model"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
	"github.com/valyala/fasthttp"
)

type BankCardController struct{}

// List 银行卡列表
func (that *BankCardController) List(ctx *fasthttp.RequestCtx) {

	banklcardNo := string(ctx.QueryArgs().Peek("card_no"))
	accounName := string(ctx.QueryArgs().Peek("real_name"))
	bankId := string(ctx.QueryArgs().Peek("bank_id"))

	ex := g.Ex{}

	if helper.CtypeDigit(banklcardNo) {
		ex["banklcard_no"] = banklcardNo
	}
	if helper.CtypeDigit(bankId) {
		ex["channel_bank_id"] = bankId
	}

	if accounName != "" {
		ex["account_name"] = accounName
	}

	data, err := model.BankCardList(ex)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

// Insert 线下卡转卡 添加银行卡
func (that *BankCardController) Insert(ctx *fasthttp.RequestCtx) {

	//fmt.Println("BankCardController Insert = ", string(ctx.PostBody()))

	bank_id := string(ctx.PostArgs().Peek("bank_id"))
	account_name := string(ctx.PostArgs().Peek("account_name"))
	//bankcard_addr := string(ctx.PostArgs().Peek("bankcard_addr"))
	banklcard_name := string(ctx.PostArgs().Peek("banklcard_name"))
	banklcard_no := string(ctx.PostArgs().Peek("banklcard_no"))

	total_max_amount := ctx.PostArgs().GetUintOrZero("total_max_amount")
	daily_max_amount := ctx.PostArgs().GetUintOrZero("daily_max_amount")

	flags := string(ctx.PostArgs().Peek("flags"))
	code := string(ctx.PostArgs().Peek("code"))
	remark := string(ctx.PostArgs().Peek("remark"))

	if !helper.CtypeDigit(bank_id) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}
	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	if !helper.CtypeDigit(banklcard_no) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if total_max_amount < 1 {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if daily_max_amount < 1 {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if flags != "2" {
		flags = "1"
	}

	// 检查该卡号是否已经存在
	_, err = model.BankCardByCol(banklcard_no)
	if err == nil {
		helper.Print(ctx, false, helper.BankCardExistErr)
		return
	}

	bc := model.Bankcard_t{

		Id:                helper.GenId(),
		ChannelBankId:     bank_id,
		BanklcardName:     banklcard_name,
		BanklcardNo:       banklcard_no,
		AccountName:       account_name,
		BankcardAddr:      "",
		State:             "0",
		Remark:            validator.FilterInjection(remark),
		DailyMaxAmount:    fmt.Sprintf("%d", daily_max_amount),
		DailyFinishAmount: "0",
		Flags:             flags,
	}

	err = model.BankCardInsert(bc, code, admin["name"])
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	//content := fmt.Sprintf("添加银行卡【卡号: %s，最大限额：%.4f, 持卡人姓名：%s】", cardNo, maxAmount, realName)
	//defer model.SystemLogWrite(content, ctx)

	helper.Print(ctx, true, helper.Success)

}

// Delete 线下卡专卡 删除银行卡
func (that *BankCardController) Delete(ctx *fasthttp.RequestCtx) {

	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	id := string(ctx.QueryArgs().Peek("id"))
	if id == "" {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	err = model.BankCardDelete(id, admin["name"])
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	// 线下转卡的paymentID  304314961990368154 刷新渠道下银行列表
	//_ = model.CacheRefreshPaymentBanks("304314961990368154")

	helper.Print(ctx, true, helper.Success)
}

// Update 编辑
func (that *BankCardController) Update(ctx *fasthttp.RequestCtx) {

	id := string(ctx.PostArgs().Peek("id"))
	state := string(ctx.PostArgs().Peek("state"))
	dailyMaxAmount := ctx.PostArgs().GetUfloatOrZero("daily_max_amount")

	//flags := string(ctx.PostArgs().Peek("flags"))
	//code := string(ctx.PostArgs().Peek("code"))
	remark := string(ctx.PostArgs().Peek("remark"))

	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	if !helper.CtypeDigit(id) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	rec := g.Record{
		"state": state,
	}

	if remark != "" {
		rec["remark"] = validator.FilterInjection(remark)
	}
	if dailyMaxAmount > 0 {
		rec["daily_max_amount"] = fmt.Sprintf("%f", dailyMaxAmount)
	}
	bankCard, err := model.BankCardByID(id)
	if err != nil {
		helper.Print(ctx, false, err)
		return
	}

	err = model.BankCardUpdate(id, rec)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	contentLog := fmt.Sprintf("渠道管理-线下银行卡-更新:后台账号:%s【银行卡名称:%s,卡号:%s,姓名:%s,当日最大入款金额:%s】",
		admin["name"], bankCard.BanklcardName, bankCard.BanklcardNo, bankCard.AccountName, bankCard.DailyMaxAmount)
	model.AdminLogInsert(model.ChannelModel, contentLog, model.UpdateOp, admin["name"])
	helper.Print(ctx, true, helper.Success)
}
