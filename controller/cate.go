package controller

import (
	"finance/contrib/helper"
	"finance/model"
	"fmt"
	"github.com/valyala/fasthttp"
)

type CateController struct{}

// List 财务管理-渠道管理-列表
func (that *CateController) List(ctx *fasthttp.RequestCtx) {

	data, err := model.CateList()
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

// UpdateState 财务管理-渠道管理-启用/停用
func (that *CateController) UpdateState(ctx *fasthttp.RequestCtx) {

	id := string(ctx.PostArgs().Peek("id"))
	state := string(ctx.PostArgs().Peek("state"))
	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	cate, err := model.CateByID(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(cate.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	if cate.State == state {
		helper.Print(ctx, false, helper.NoDataUpdate)
		return
	}

	err = model.CateSet(id, state)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	contentLog := fmt.Sprintf("财务管理-渠道管理-%s:后台账号:%s【商户ID: %s ；渠道名称: %s,id:%s,状态:%s】",
		model.StateMap[state], admin["name"], cate.MerchantId, cate.Name, id, model.StateMap[state])
	model.AdminLogInsert(model.ChannelModel, contentLog, model.DeleteOp, admin["name"])

	helper.Print(ctx, true, helper.Success)
}

func (that *CateController) Cache(ctx *fasthttp.RequestCtx) {

	data := model.CateListRedis()
	helper.PrintJson(ctx, true, data)
}

// Withdraw 财务管理-提款通道
func (that *CateController) Withdraw(ctx *fasthttp.RequestCtx) {

	amount := ctx.PostArgs().GetUfloatOrZero("amount")

	data, err := model.CateWithdrawList(amount)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}
