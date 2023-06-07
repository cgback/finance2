package controller

import (
	"finance/contrib/helper"
	"finance/contrib/validator"
	"finance/model"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
	"github.com/valyala/fasthttp"
	"strconv"
)

type UsdtController struct{}

func (that *UsdtController) Info(ctx *fasthttp.RequestCtx) {

	res, err := model.UsdtInfo()
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, res)
}

func (that *UsdtController) Update(ctx *fasthttp.RequestCtx) {

	depositUsdtRate := string(ctx.PostArgs().Peek("deposit_usdt_rate"))
	withdrawUsdtRate := string(ctx.PostArgs().Peek("withdraw_usdt_rate"))
	code := string(ctx.PostArgs().Peek("code"))
	_, err := strconv.ParseFloat(depositUsdtRate, 64)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}
	_, err = strconv.ParseFloat(withdrawUsdtRate, 64)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}
	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	if !helper.CtypeDigit(code) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	err = model.UsdtUpdate(depositUsdtRate, withdrawUsdtRate, admin["name"])
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// USDT 发起线下USDT
func (that *UsdtController) Pay(ctx *fasthttp.RequestCtx) {

	amount := string(ctx.PostArgs().Peek("amount"))
	rate := string(ctx.PostArgs().Peek("rate"))
	id := string(ctx.PostArgs().Peek("id"))
	addr := string(ctx.PostArgs().Peek("addr"))
	protocolType := string(ctx.PostArgs().Peek("protocol_type"))
	hashID := string(ctx.PostArgs().Peek("hash_id"))

	if protocolType != "TRC20" || addr == "" {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	//if len(hashID) != 64 || len(hashID) != 32 || len(hashID) != 128 || len(hashID) != 256 {
	//	helper.Print(ctx, false, helper.InvalidTransactionHash)
	//	return
	//}

	if id != "779402438062874469" {
		helper.Print(ctx, false, helper.ChannelIDErr)
		return
	}

	if !helper.CtypeDigit(amount) {
		helper.Print(ctx, false, helper.AmountErr)
		return
	}

	if addr == "" {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	order, err := model.UsdtPay(ctx, id, amount, rate, addr, protocolType, hashID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}
	msg := map[string]string{}
	msg["order_id"] = order

	helper.Print(ctx, true, msg)
}

// Insert 线下usdt 添加usdt收款账号
func (that *UsdtController) Insert(ctx *fasthttp.RequestCtx) {

	name := string(ctx.PostArgs().Peek("name"))
	addr := string(ctx.PostArgs().Peek("wallet_addr"))
	qrImg := string(ctx.PostArgs().Peek("qr_img"))
	maxAmount := ctx.PostArgs().GetUfloatOrZero("max_amount")
	minAmount := ctx.PostArgs().GetUfloatOrZero("min_amount")
	state := string(ctx.PostArgs().Peek("state"))
	sort := ctx.PostArgs().GetUintOrZero("sort")
	remark := string(ctx.PostArgs().Peek("remark"))
	code := string(ctx.PostArgs().Peek("code"))

	if len(name) == 0 || len(addr) == 0 || maxAmount <= 0 || minAmount <= 0 || len(state) == 0 {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}
	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	if state != "2" {
		state = "1"
	}

	// 检查该卡号是否已经存在
	_, err = model.UsdtByCol(addr)
	if err == nil {
		helper.Print(ctx, false, helper.VirtualWalletAddressExist)
		return
	}

	vw := model.VirtualWallet_t{
		Id:          helper.GenId(),
		Name:        name,
		Currency:    "1",
		Pid:         "779402438062874469",
		Protocol:    "1",
		WalletAddr:  addr,
		State:       state,
		Remark:      validator.FilterInjection(remark),
		MaxAmount:   maxAmount,
		MinAmount:   minAmount,
		QrImg:       qrImg,
		Sort:        sort,
		CreatedAt:   ctx.Time().Unix(),
		CreatedName: admin["name"],
		CreatedUID:  admin["id"],
		UpdatedAt:   ctx.Time().Unix(),
		UpdatedName: admin["name"],
		UpdatedUID:  admin["id"],
	}

	err = model.UsdtInsert(vw, code, admin["name"])
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)

}

// List usdt列表
func (that *UsdtController) List(ctx *fasthttp.RequestCtx) {

	page := ctx.PostArgs().GetUintOrZero("page")
	pageSize := ctx.PostArgs().GetUintOrZero("page_size")
	fmt.Println(page)
	if page < 1 {
		page = 1
	}
	if pageSize < 10 {
		pageSize = 10
	}
	data, err := model.VirtualWalletList(g.Ex{}, uint(page), uint(pageSize))
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

// Withdraw 会员申请提现
func (that *UsdtController) Withdraw(ctx *fasthttp.RequestCtx) {

	rate := string(ctx.PostArgs().Peek("rate"))
	amount := string(ctx.PostArgs().Peek("amount"))
	fmt.Println(rate, amount)
	id, err := model.UsdtWithdrawUserInsert(amount, rate, ctx)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, id)
}

// Delete  删除usdt
func (that *UsdtController) Delete(ctx *fasthttp.RequestCtx) {

	id := string(ctx.QueryArgs().Peek("id"))
	if id == "" {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}
	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	err = model.UsdtDelete(id, admin["name"])
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// Update 编辑
func (that *UsdtController) UpdateAccount(ctx *fasthttp.RequestCtx) {

	id := string(ctx.PostArgs().Peek("id"))
	name := string(ctx.PostArgs().Peek("name"))
	addr := string(ctx.PostArgs().Peek("wallet_addr"))
	qrImg := string(ctx.PostArgs().Peek("qr_img"))
	maxAmount := ctx.PostArgs().GetUfloatOrZero("max_amount")
	minAmount := ctx.PostArgs().GetUfloatOrZero("min_amount")
	state := string(ctx.PostArgs().Peek("state"))
	sort := ctx.PostArgs().GetUintOrZero("sort")
	remark := string(ctx.PostArgs().Peek("remark"))
	code := string(ctx.PostArgs().Peek("code"))
	if id == "" {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}
	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}
	if len(code) == 0 {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if !helper.CtypeDigit(id) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}
	if len(name) == 0 || len(addr) == 0 || maxAmount <= 0 || minAmount <= 0 || len(state) == 0 {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}
	rec := g.Record{
		"state": state,
	}

	if remark != "" {
		rec["remark"] = validator.FilterInjection(remark)
	}
	if maxAmount > 0 {
		rec["max_amount"] = fmt.Sprintf("%f", maxAmount)
	}
	if minAmount > 0 {
		rec["min_amount"] = fmt.Sprintf("%f", minAmount)
	}
	if len(qrImg) > 0 {
		rec["qr_img"] = qrImg
	}
	rec["sort"] = sort
	rec["remark"] = remark
	vw, err := model.UsdtByID(id)
	if err != nil {
		helper.Print(ctx, false, err)
		return
	}
	rec["updated_at"] = ctx.Time().Unix()
	rec["updated_name"] = admin["name"]
	rec["updated_uid"] = admin["id"]
	if state == "1" {
		d, err := model.VirtualWalletList(g.Ex{"state": 1, "id": g.Op{"neq": id}}, 1, 10)
		if err != nil {
			helper.Print(ctx, false, err.Error())
			return
		}
		if d.T > 0 {
			helper.Print(ctx, false, helper.CanOnlyOpenOnePayeeAccount)
			return
		}
	}
	err = model.VirtualWalletUpdate(id, rec)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	contentLog := fmt.Sprintf("渠道管理-线下USDT-更新:后台账号:%s【id:%s,USDT名称:%s,地址:%s,最大:%f,最小:%f,二维码:%s,排序:%d,备注：%s】",
		admin["name"], vw.Id, vw.Name, vw.WalletAddr, vw.MaxAmount, vw.MinAmount, vw.QrImg, vw.Sort, vw.Remark)
	model.AdminLogInsert(model.ChannelModel, contentLog, model.UpdateOp, admin["name"])
	helper.Print(ctx, true, helper.Success)
}

// Update 编辑
func (that *UsdtController) UpdateState(ctx *fasthttp.RequestCtx) {

	id := string(ctx.PostArgs().Peek("id"))
	state := string(ctx.PostArgs().Peek("state"))

	if id == "" {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}
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

	vw, err := model.UsdtByID(id)
	if err != nil {
		helper.Print(ctx, false, err)
		return
	}
	if state == "1" {
		d, err := model.VirtualWalletList(g.Ex{"state": 1, "id": g.Op{"neq": id}}, 1, 10)
		if err != nil {
			helper.Print(ctx, false, err.Error())
			return
		}
		if d.T > 0 {
			helper.Print(ctx, false, helper.CanOnlyOpenOnePayeeAccount)
			return
		}
	}
	rec["updated_at"] = ctx.Time().Unix()
	rec["updated_name"] = admin["name"]
	rec["updated_uid"] = admin["id"]
	err = model.VirtualWalletUpdate(id, rec)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	contentLog := fmt.Sprintf("渠道管理-线下USDT-更新状态:后台账号:%s【id:%s,USDT名称:%s,地址:%s,最大:%f,最小:%f,二维码:%s,排序:%d,备注：%s】",
		admin["name"], vw.Id, vw.Name, vw.WalletAddr, vw.MaxAmount, vw.MinAmount, vw.QrImg, vw.Sort, vw.Remark)
	model.AdminLogInsert(model.ChannelModel, contentLog, model.UpdateOp, admin["name"])
	helper.Print(ctx, true, helper.Success)
}
