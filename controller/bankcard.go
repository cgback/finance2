package controller

import (
	"finance/contrib/helper"
	"finance/contrib/validator"
	"finance/model"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
	"github.com/valyala/fasthttp"
	"strconv"
	"strings"
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

	bankId := string(ctx.PostArgs().Peek("bank_id"))
	accountName := string(ctx.PostArgs().Peek("account_name"))
	banklcardName := string(ctx.PostArgs().Peek("banklcard_name"))
	banklcardNo := string(ctx.PostArgs().Peek("banklcard_no"))
	dailyMaxAmount := ctx.PostArgs().GetUintOrZero("daily_max_amount")
	fmin := string(ctx.PostArgs().Peek("fmin"))
	fmax := string(ctx.PostArgs().Peek("fmax"))
	flags := string(ctx.PostArgs().Peek("flags")) //1转卡 2转账
	code := string(ctx.PostArgs().Peek("code"))
	remark := string(ctx.PostArgs().Peek("remark"))
	vips := string(ctx.PostArgs().Peek("vip_list"))
	amountList := string(ctx.PostArgs().Peek("amount_list"))
	discount := string(ctx.PostArgs().Peek("discount"))
	isZone := ctx.PostArgs().GetUintOrZero("is_zone")
	isFast := ctx.PostArgs().GetUintOrZero("is_fast")
	cid := ctx.PostArgs().GetUintOrZero("cid")

	if bankId != "" {
		if !helper.CtypeDigit(bankId) {
			helper.Print(ctx, false, helper.ParamErr)
			return
		}
	}
	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	if !helper.CtypeDigit(banklcardNo) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if dailyMaxAmount < 1 {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if flags != "2" {
		flags = "1"
	}

	if vips != "" {
		vipSlice := strings.Split(vips, ",")
		for _, v := range vipSlice {
			if !validator.CheckStringDigit(v) || !validator.CheckIntScope(v, 1, 11) {
				helper.Print(ctx, false, helper.MemberLevelErr)
				return
			}
		}
	}

	if amountList != "" {
		amountSlice := strings.Split(amountList, ",")
		for _, v := range amountSlice {
			if !validator.CheckStringDigit(v) {
				helper.Print(ctx, false, helper.AmountErr)
				return
			}
		}
	}

	if fmin != "" && fmax != "" {
		if !validator.CheckStringDigit(fmin) || !validator.CheckStringDigit(fmax) {
			helper.Print(ctx, false, helper.AmountErr)
			return
		}

		minAmountInt, _ := strconv.Atoi(fmin)
		maxAmountInt, _ := strconv.Atoi(fmax)
		if minAmountInt > maxAmountInt {
			helper.Print(ctx, false, helper.AmountErr)
			return
		}
	}

	if discount != "" {
		if !validator.CheckStringDigit(discount) {
			helper.Print(ctx, false, helper.AmountErr)
			return
		}
	}

	// 检查该卡号是否已经存在
	_, err = model.BankCardByCol(banklcardNo)
	if err == nil {
		helper.Print(ctx, false, helper.BankCardExistErr)
		return
	}

	bc := model.Bankcard_t{

		Id:                helper.GenId(),
		ChannelBankId:     bankId,
		BanklcardName:     banklcardName,
		BanklcardNo:       banklcardNo,
		AccountName:       accountName,
		BankcardAddr:      "",
		State:             "0",
		Remark:            validator.FilterInjection(remark),
		DailyMaxAmount:    fmt.Sprintf("%d", dailyMaxAmount),
		DailyFinishAmount: "0",
		Flags:             flags,
		VipList:           vips,
		Fmin:              fmin,
		Fmax:              fmax,
		AmountList:        amountList,
		Discount:          discount,
		IsZone:            isZone,
		IsFast:            isFast,
		Cid:               cid,
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
	remark := string(ctx.PostArgs().Peek("remark"))
	fmin := string(ctx.PostArgs().Peek("fmin"))
	fmax := string(ctx.PostArgs().Peek("fmax"))
	vips := string(ctx.PostArgs().Peek("vip_list"))
	amountList := string(ctx.PostArgs().Peek("amount_list"))
	discount := string(ctx.PostArgs().Peek("discount"))
	isZone := ctx.PostArgs().GetUintOrZero("is_zone")
	isFast := ctx.PostArgs().GetUintOrZero("is_fast")

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

	if vips != "" {
		vipSlice := strings.Split(vips, ",")
		for _, v := range vipSlice {
			if !validator.CheckStringDigit(v) || !validator.CheckIntScope(v, 1, 11) {
				helper.Print(ctx, false, helper.MemberLevelErr)
				return
			}
		}
		rec["vip_list"] = vips
	}

	if amountList != "" {
		amountSlice := strings.Split(amountList, ",")
		for _, v := range amountSlice {
			if !validator.CheckStringDigit(v) {
				helper.Print(ctx, false, helper.AmountErr)
				return
			}
		}
		rec["amount_list"] = amountList
	}

	if fmin != "" && fmax != "" {
		if !validator.CheckStringDigit(fmin) || !validator.CheckStringDigit(fmax) {
			helper.Print(ctx, false, helper.AmountErr)
			return
		}

		minAmountInt, _ := strconv.Atoi(fmin)
		maxAmountInt, _ := strconv.Atoi(fmax)
		if minAmountInt > maxAmountInt {
			helper.Print(ctx, false, helper.AmountErr)
			return
		}
		rec["fmax"] = fmax
		rec["fmin"] = fmin
	}

	if discount != "" {
		if !validator.CheckStringDigit(discount) {
			helper.Print(ctx, false, helper.AmountErr)
			return
		}
		rec["discount"] = discount
	}
	if remark != "" {
		rec["remark"] = validator.FilterInjection(remark)
	}
	if dailyMaxAmount > 0 {
		rec["daily_max_amount"] = fmt.Sprintf("%f", dailyMaxAmount)
	}
	rec["is_zone"] = isZone
	rec["is_fast"] = isFast
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
