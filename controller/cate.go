package controller

import (
	"finance/contrib/helper"
	"finance/contrib/validator"
	"finance/model"
	"fmt"
	"github.com/valyala/fasthttp"
)

type CateController struct{}

type cateListParam struct {
	All      string `rule:"digit" min:"0" max:"1" default:"0" msg:"all error" name:"all"` // 商户id
	CateName string `rule:"none" msg:"cate_name error" name:"cate_name"`                  // 渠道名称
}

type cateStateParam struct {
	ID    string `rule:"digit" default:"0" msg:"id error" name:"id"`
	State string `rule:"digit" min:"0" max:"1" msg:"state error" name:"state"` // 0:关闭1:开启
	Code  string `rule:"digit" msg:"code error" name:"code"`                   // 动态验证码
}

// List 财务管理-渠道管理-列表
func (that *CateController) List(ctx *fasthttp.RequestCtx) {

	param := cateListParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if param.CateName != "" {
		if !validator.CheckStringCHNAlnum(param.CateName) || !validator.CheckStringLength(param.CateName, 1, 20) {
			helper.Print(ctx, false, helper.CateNameErr)
			return
		}
	}

	data, err := model.CateList(param.CateName, param.All)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

// UpdateState 财务管理-渠道管理-启用/停用
func (that *CateController) UpdateState(ctx *fasthttp.RequestCtx) {

	param := cateStateParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	cate, err := model.CateByID(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(cate.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	if cate.State == param.State {
		helper.Print(ctx, false, helper.NoDataUpdate)
		return
	}

	/*
		keyword := "开启"
		if param.State == "0" {
			keyword = "关闭"
		}
		content := fmt.Sprintf("%s【商户ID: %s ；渠道名称: %s】", keyword, cate.MerchantId, cate.Name)
		defer model.SystemLogWrite(content, ctx)
	*/
	err = model.CateSet(param.ID, param.State)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	contentLog := fmt.Sprintf("财务管理-渠道管理-%s:后台账号:%s【商户ID: %s ；渠道名称: %s,id:%s,状态:%s】",
		model.StateMap[param.State], admin["name"], cate.MerchantId, cate.Name, param.ID, model.StateMap[param.State])
	model.AdminLogInsert(model.ChannelModel, contentLog, model.DeleteOp, admin["name"])

	helper.Print(ctx, true, helper.Success)
}
