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
	state := string(ctx.QueryArgs().Peek("state"))
	vip := string(ctx.QueryArgs().Peek("vip"))
	cid := string(ctx.QueryArgs().Peek("cid"))
	flags := string(ctx.QueryArgs().Peek("flags"))

	ex := g.Ex{}

	if helper.CtypeDigit(banklcardNo) {
		ex["banklcard_no"] = banklcardNo
	}
	if helper.CtypeDigit(state) {
		ex["state"] = state
	}
	if helper.CtypeDigit(state) {
		ex["state"] = state
	}
	if helper.CtypeDigit(cid) {
		ex["cid"] = cid
	}
	if helper.CtypeDigit(flags) {
		ex["flags"] = flags
	}

	if accounName != "" {
		ex["account_name"] = accounName
	}

	data, err := model.BankCardList(ex, vip)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

// Insert 线下卡转卡 添加银行卡
func (that *BankCardController) Insert(ctx *fasthttp.RequestCtx) {

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
	seq := ctx.PostArgs().GetUintOrZero("seq")
	paymentName := string(ctx.PostArgs().Peek("payment_name"))

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
		Cid:               int64(cid),
		CreatedAt:         ctx.Time().Unix(),
		CreatedUID:        admin["id"],
		CreatedName:       admin["name"],
		Seq:               seq,
	}

	err = model.BankCardInsert(bc, code, admin["name"])
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	fields := map[string]string{
		"payment_name": paymentName,
	}
	if cid == 2 && flags == "1" {
		fields["id"] = "766870294997073617"
	}
	if cid == 2 && flags == "2" {
		fields["id"] = "766870294997073618"
	}
	if cid == 4 && flags == "1" {
		fields["id"] = "766870294997073619"
	}
	if cid == 4 && flags == "2" {
		fields["id"] = "766870294997073620"
	}
	if cid == 3 && flags == "1" {
		fields["id"] = "766870294997073621"
	}

	fields["updated_at"] = fmt.Sprintf(`%d`, ctx.Time().Unix())
	fields["updated_uid"] = admin["id"]
	fields["updated_name"] = admin["name"]
	err = model.ChannelUpdatePaymentName(fields)
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
	seq := ctx.PostArgs().GetUintOrZero("seq")
	paymentName := string(ctx.PostArgs().Peek("payment_name"))
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
	if seq != 0 {
		rec["seq"] = seq
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
		if !validator.CheckFloat(discount) {
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
	rec["updated_at"] = ctx.Time().Unix()
	rec["updated_uid"] = admin["id"]
	rec["updated_name"] = admin["name"]
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

	fields := map[string]string{
		"payment_name": paymentName,
	}
	if bankCard.Cid == 2 && bankCard.Flags == "1" {
		fields["id"] = "766870294997073617"
	}
	if bankCard.Cid == 2 && bankCard.Flags == "2" {
		fields["id"] = "766870294997073618"
	}
	if bankCard.Cid == 4 && bankCard.Flags == "1" {
		fields["id"] = "766870294997073619"
	}
	if bankCard.Cid == 4 && bankCard.Flags == "2" {
		fields["id"] = "766870294997073620"
	}
	if bankCard.Cid == 3 && bankCard.Flags == "1" {
		fields["id"] = "766870294997073621"
	}

	fields["updated_at"] = fmt.Sprintf(`%d`, ctx.Time().Unix())
	fields["updated_uid"] = admin["id"]
	fields["updated_name"] = admin["name"]
	err = model.ChannelUpdatePaymentName(fields)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	contentLog := fmt.Sprintf("渠道管理-线下银行卡-更新:后台账号:%s【银行卡名称:%s,卡号:%s,姓名:%s,当日最大入款金额:%s】",
		admin["name"], bankCard.BanklcardName, bankCard.BanklcardNo, bankCard.AccountName, bankCard.DailyMaxAmount)
	model.AdminLogInsert(model.ChannelModel, contentLog, model.UpdateOp, admin["name"])
	helper.Print(ctx, true, helper.Success)
}

func (that *BankCardController) InsertMsg(ctx *fasthttp.RequestCtx) {

	cid := string(ctx.PostArgs().Peek("cid"))        //1QR Banking 2MomoPay 3ZaloPay 4ViettelPay 5Thẻ Cào 6Offline 7USDT
	flags := string(ctx.PostArgs().Peek("flags"))    //1转卡 2转账
	h5img := string(ctx.PostArgs().Peek("h5_img"))   //h5图片
	webimg := string(ctx.PostArgs().Peek("web_img")) //web 图片
	appimg := string(ctx.PostArgs().Peek("app_img")) //app 图片

	fields := map[string]string{
		"h5_img":  h5img,
		"web_img": webimg,
		"app_img": appimg,
	}
	if cid == "2" && flags == "1" {
		fields["id"] = "766870294997073617"
	}
	if cid == "2" && flags == "2" {
		fields["id"] = "766870294997073618"
	}
	if cid == "4" && flags == "1" {
		fields["id"] = "766870294997073619"
	}
	if cid == "4" && flags == "2" {
		fields["id"] = "766870294997073620"
	}
	if cid == "3" && flags == "1" {
		fields["id"] = "766870294997073621"
	}
	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}
	fields["updated_at"] = fmt.Sprintf(`%d`, ctx.Time().Unix())
	fields["updated_uid"] = admin["id"]
	fields["updated_name"] = admin["name"]
	err = model.ChannelUpdateImg(fields)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}
	helper.Print(ctx, true, helper.Success)

}

func (that *BankCardController) UpdateDiscount(ctx *fasthttp.RequestCtx) {

	cid := string(ctx.PostArgs().Peek("cid"))           //1QR Banking 2MomoPay 3ZaloPay 4ViettelPay 5Thẻ Cào 6Offline 7USDT
	flags := string(ctx.PostArgs().Peek("flags"))       //1转卡 2转账
	discount := string(ctx.PostArgs().Peek("discount")) //优惠

	fields := map[string]string{
		"discount": discount,
	}
	if cid == "2" && flags == "1" {
		fields["id"] = "766870294997073617"
	}
	if cid == "2" && flags == "2" {
		fields["id"] = "766870294997073618"
	}
	if cid == "4" && flags == "1" {
		fields["id"] = "766870294997073619"
	}
	if cid == "4" && flags == "2" {
		fields["id"] = "766870294997073620"
	}
	if cid == "3" && flags == "1" {
		fields["id"] = "766870294997073621"
	}
	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}
	fields["updated_at"] = fmt.Sprintf(`%d`, ctx.Time().Unix())
	fields["updated_uid"] = admin["id"]
	fields["updated_name"] = admin["name"]
	err = model.ChannelUpdateDiscount(fields)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}
	helper.Print(ctx, true, helper.Success)

}
