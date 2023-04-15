package controller

import (
	"finance/contrib/helper"
	"finance/model"
	"github.com/valyala/fasthttp"
)

type LogController struct{}

// 日志列表
func (that LogController) List(ctx *fasthttp.RequestCtx) {

	s, err := model.LogList()
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.PrintJson(ctx, true, s)
}
