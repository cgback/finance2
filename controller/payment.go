package controller

import (
	"finance/contrib/helper"
	"finance/contrib/validator"
	"finance/model"
	"fmt"
	"github.com/valyala/fasthttp"
)

type PaymentController struct{}

type updatePaymentParam struct {
	ID          string `rule:"digit" msg:"id error" name:"id"`
	Name        string `rule:"none" msg:"name error" name:"name"`                   // 通道名称
	State       string `rule:"none" msg:"state error" name:"state"`                 //状态
	Sort        string `rule:"digit" min:"1" max:"99" msg:"sort error" name:"sort"` // 排序
	Comment     string `rule:"none" msg:"comment error" name:"comment"`             // 备注
	AmountList  string `rule:"none" msg:"amount_list error" name:"amount_list"`     // 快捷金额列表
	VipList     string `rule:"vip_list" msg:"vip_list error" name:"vip_list"`       //会员等级
	Discount    string `rule:"float" msg:"discount error" name:"discount"`          //优惠
	WebImg      string `rule:"none" msg:"web_img error" name:"web_img"`             //web端说明
	H5Img       string `rule:"none" msg:"h5_img error" name:"h5_img"`               //h5端说明
	AppImg      string `rule:"none" msg:"app_img error" name:"app_img"`             //app端说明
	Code        string `rule:"digit" msg:"code error" name:"code"`                  // 动态验证码
	Fmax        string `rule:"digit" msg:"fmax" name:"fmax"`
	Fmin        string `rule:"digit" msg:"fmin" name:"fmin"`
	PaymentName string `rule:"none" msg:"payment_name error" name:"payment_name"`
	IsZone      string `rule:"none" msg:"is_zone error" name:"is_zone"`
	IsFast      string `rule:"none" msg:"is_fast error" name:"is_fast"`
}

type chanStateParam struct {
	ID    string `rule:"digit" default:"0" msg:"id error" name:"id"`
	State string `rule:"digit" min:"0" max:"1" msg:"state error" name:"state"` // 0:关闭1:开启
	Code  string `rule:"digit" msg:"code error" name:"code"`                   // 动态验证码
}

// List 财务管理-渠道管理-通道管理-列表
func (that *PaymentController) List(ctx *fasthttp.RequestCtx) {

	var cateId string
	channelName := string(ctx.PostArgs().Peek("channel_name"))
	vip := string(ctx.PostArgs().Peek("vip"))
	state := string(ctx.PostArgs().Peek("state"))
	flag := string(ctx.PostArgs().Peek("flag"))
	paymentName := string(ctx.PostArgs().Peek("payment_name"))
	name := string(ctx.PostArgs().Peek("name"))
	comment := string(ctx.PostArgs().Peek("comment"))

	cateName := string(ctx.PostArgs().Peek("cate_name"))
	fmt.Println("cateName:", cateName)
	if cateName != "" {
		cate, err := model.CateListByName(cateName)
		if err == nil {
			fmt.Println(cate.ID)
			cateId = cate.ID
		} else {
			helper.Print(ctx, false, err.Error())
			return
		}
	}

	data, err := model.PaymentList(cateId, channelName, vip, state, flag, paymentName, name, comment)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

// Update 财务管理-渠道管理-通道管理-修改
func (that *PaymentController) Update(ctx *fasthttp.RequestCtx) {

	param := updatePaymentParam{}
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

	if param.Comment != "" {
		if !validator.CheckStringLength(param.Comment, 0, 50) {
			helper.Print(ctx, false, helper.RemarkFMTErr)
			return
		}
	}

	if param.AmountList != "" {
		if !validator.CheckStringCommaDigit(param.AmountList) {
			helper.Print(ctx, false, helper.AmountErr)
			return
		}
	}

	// 校验渠道id和通道id是否存在
	payment, err := model.ChanExistsByID(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(payment.ID) == 0 {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	// 三方渠道
	cate, err := model.CateByID(payment.CateID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(cate.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	// 三方通道
	channel, err := model.TunnelByID(payment.ChannelID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(channel.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	fields := map[string]string{
		"id":           param.ID,
		"quota":        "0",
		"sort":         param.Sort,
		"comment":      param.Comment,
		"amount_list":  param.AmountList,
		"fmax":         param.Fmax,
		"fmin":         param.Fmin,
		"vip_list":     param.VipList,
		"payment_name": param.PaymentName,
		"is_zone":      param.IsZone,
		"is_fast":      param.IsFast,
		"h5_img":       param.H5Img,
		"web_img":      param.WebImg,
		"app_img":      param.AppImg,
		"discount":     param.Discount,
	}

	if len(param.AmountList) > 0 {
		//TODO RPC
	}

	err = model.ChannelUpdate(fields)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if param.State == "1" && channel.State != "1" {
		fmt.Println("通道开启了也去开启支付方式")
		model.ChannelTypeUpdateState(channel.ID, "1", admin)
	}

	contentLog := fmt.Sprintf("财务管理-渠道管理-通道管理-修改:后台账号:%s【渠道名:%s;通道名:%s;子通道名:%s=>%s;最小金额:%s=>%s；最大金额:%s=>%s,金额:%s=>%s】",
		admin["name"], cate.Name, channel.Name, payment.PaymentName, fields["payment_name"], payment.Fmin, fields["fmin"],
		payment.Fmax, fields["fmax"], payment.AmountList, fields["amount_list"])
	model.AdminLogInsert(model.ChannelModel, contentLog, model.UpdateOp, admin["name"])

	helper.Print(ctx, true, helper.Success)
}

// UpdateState 财务管理-渠道管理-通道管理-启用/停用
func (that *PaymentController) UpdateState(ctx *fasthttp.RequestCtx) {

	id := string(ctx.PostArgs().Peek("id"))
	state := string(ctx.PostArgs().Peek("state"))

	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	// 校验渠道id和通道id是否存在
	payment, err := model.ChanExistsByID(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(payment.ID) == 0 {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	if payment.State == state {
		helper.Print(ctx, false, helper.NoDataUpdate)
		return
	}

	// 三方渠道
	cate, err := model.CateByID(payment.CateID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(cate.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	// 三方通道
	channel, err := model.TunnelByID(payment.ChannelID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(channel.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	err = model.ChannelSet(id, state, admin["id"], admin["name"])
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	fmt.Println(state, ":", channel.State)
	if state == "1" && channel.State != "1" {
		fmt.Println("通道开启了也去开启支付方式")
		model.ChannelTypeUpdateState(channel.ID, "1", admin)
	}

	contentLog := fmt.Sprintf(" 财务管理-渠道管理-通道管理-%s:后台账号:%s【渠道名称: %s ；通道名称: %s,id:%s】",
		model.StateMap[state], admin["name"], cate.Name, channel.Name, id)
	model.AdminLogInsert(model.ChannelModel, contentLog, model.StateMap[state], admin["name"])

	helper.Print(ctx, true, helper.Success)
}
