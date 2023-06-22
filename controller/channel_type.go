package controller

import (
	"finance/contrib/helper"
	"finance/model"
	"github.com/valyala/fasthttp"
)

type ChannelTypeController struct{}

// 日志列表
func (that ChannelTypeController) List(ctx *fasthttp.RequestCtx) {

	s, err := model.ChannelTypeList()
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, s)
}

func (that ChannelTypeController) UpdateState(ctx *fasthttp.RequestCtx) {

	id := string(ctx.QueryArgs().Peek("id"))
	state := string(ctx.QueryArgs().Peek("state"))
	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}
	err = model.ChannelTypeUpdateState(id, state, admin)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

func (that ChannelTypeController) UpdateSort(ctx *fasthttp.RequestCtx) {

	id := string(ctx.QueryArgs().Peek("id"))
	sort := ctx.QueryArgs().GetUintOrZero("sort")

	err := model.ChannelTypeUpdateSort(id, sort)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}
