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
	ID         string `rule:"digit" msg:"id error" name:"id"`
	Name       string `rule:"none" msg:"name error" name:"name"`                   // 通道名称
	State      string `rule:"none" msg:"state error" name:"state"`                 //状态
	Sort       string `rule:"digit" min:"1" max:"99" msg:"sort error" name:"sort"` // 排序
	Comment    string `rule:"none" msg:"comment error" name:"comment"`             // 备注
	AmountList string `rule:"none" msg:"amount_list error" name:"amount_list"`     // 快捷金额列表
	VipList    string `rule:"vip_list" msg:"vip_list error" name:"vip_list"`       //会员等级
	Discount   string `rule:"float" msg:"discount error" name:"discount"`          //优惠
	WebImg     string `rule:"none" msg:"web_img error" name:"web_img"`             //web端说明
	H5Img      string `rule:"none" msg:"h5_img error" name:"h5_img"`               //h5端说明
	AppImg     string `rule:"none" msg:"app_img error" name:"app_img"`             //app端说明
	Code       string `rule:"digit" msg:"code error" name:"code"`                  // 动态验证码
}

type paymentListParam struct {
	CateID    string `rule:"digit" default:"0" msg:"cate_id error" name:"cate_id"`       // 渠道id
	ChannelID string `rule:"digit" default:"0" msg:"channel_id error" name:"channel_id"` // 通道id
	St        string `rule:"time" msg:"st error" name:"st"`                              // 开始时间
	Et        string `rule:"time" msg:"et error" name:"et"`                              // 结束时间
	Vip       string `rule:"digit" default:"0" msg:"vip error" name:"vip"`               //vip等级
	State     string `rule:"none" default:"" msg:"state error" name:"state"`             // 0关闭1开启
	Flag      string `rule:"digit" default:"0" msg:"flag error" name:"flag"`             //1充值2提现
}

type chanStateParam struct {
	ID    string `rule:"digit" default:"0" msg:"id error" name:"id"`
	State string `rule:"digit" min:"0" max:"1" msg:"state error" name:"state"` // 0:关闭1:开启
	Code  string `rule:"digit" msg:"code error" name:"code"`                   // 动态验证码
}

// List 财务管理-渠道管理-通道管理-列表
func (that *PaymentController) List(ctx *fasthttp.RequestCtx) {

	param := paymentListParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	data, err := model.PaymentList(param.CateID, param.ChannelID)
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
		"id":          param.ID,
		"quota":       "0",
		"sort":        param.Sort,
		"comment":     param.Comment,
		"amount_list": param.AmountList,
	}

	if len(param.AmountList) > 0 {
		//TODO RPC
		//min, err := decimal.NewFromString(param.FMin)
		//if err != nil {
		//	helper.Print(ctx, false, helper.AmountErr)
		//	return
		//}
		//max, err := decimal.NewFromString(param.FMax)
		//if err != nil {
		//	helper.Print(ctx, false, helper.AmountErr)
		//	return
		//}
		//if strings.Contains(param.AmountList, ",") {
		//	list := strings.Split(param.AmountList, ",")
		//	for _, v := range list {
		//		amount, err := decimal.NewFromString(v)
		//		if err != nil {
		//			helper.Print(ctx, false, helper.AmountErr)
		//			return
		//		}
		//		if amount.LessThan(min) {
		//			helper.Print(ctx, false, helper.AmountErr)
		//			return
		//		}
		//		if amount.GreaterThan(max) {
		//			helper.Print(ctx, false, helper.AmountErr)
		//			return
		//		}
		//	}
		//} else {
		//	amount, err := decimal.NewFromString(param.AmountList)
		//	if err != nil {
		//		helper.Print(ctx, false, helper.AmountErr)
		//		return
		//	}
		//	if amount.LessThan(min) {
		//		helper.Print(ctx, false, helper.AmountErr)
		//		return
		//	}
		//	if amount.GreaterThan(max) {
		//		helper.Print(ctx, false, helper.AmountErr)
		//		return
		//	}
		//}
	}

	err = model.ChannelUpdate(fields)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	contentLog := fmt.Sprintf("财务管理-渠道管理-通道管理-修改:后台账号:%s【渠道名:%s;通道名:%s;子通道名:%s=>%s;最小金额:%s=>%s；最大金额:%s=>%s,金额:%s=>%s】",
		admin["name"], cate.Name, channel.Name, payment.PaymentName, fields["payment_name"], payment.Fmin, fields["fmin"],
		payment.Fmax, fields["fmax"], payment.AmountList, fields["amount_list"])
	model.AdminLogInsert(model.ChannelModel, contentLog, model.UpdateOp, admin["name"])

	helper.Print(ctx, true, helper.Success)
}

// UpdateState 财务管理-渠道管理-通道管理-启用/停用
func (that *PaymentController) UpdateState(ctx *fasthttp.RequestCtx) {

	param := chanStateParam{}
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

	if payment.State == param.State {
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

	// 上级渠道关闭的时候不能开启
	if param.State == "1" && cate.State == "0" {
		helper.Print(ctx, false, helper.ParentChannelClosed)
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

	err = model.ChannelSet(param.ID, param.State)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	contentLog := fmt.Sprintf(" 财务管理-渠道管理-通道管理-%s:后台账号:%s【渠道名称: %s ；通道名称: %s,id:%s】",
		model.StateMap[param.State], admin["name"], cate.Name, channel.Name, param.ID)
	model.AdminLogInsert(model.ChannelModel, contentLog, model.StateMap[param.State], admin["name"])

	helper.Print(ctx, true, helper.Success)
}
