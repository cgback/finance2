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
