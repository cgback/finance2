package controller

import (
	"finance/contrib/helper"
	"finance/model"
	"github.com/valyala/fasthttp"
	"strconv"
)

type ConfigController struct{}

func (that *ConfigController) Detail(ctx *fasthttp.RequestCtx) {

	res, err := model.ConfigDetail()
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, res)
}

func (that *ConfigController) Deposit(ctx *fasthttp.RequestCtx) {

	depositTimeSwitch := string(ctx.PostArgs().Peek("deposit_time_switch"))
	depositTimeOneMin := string(ctx.PostArgs().Peek("deposit_time_one_min"))
	depositTimeOneMax := string(ctx.PostArgs().Peek("deposit_time_one_max"))
	depositTimeTwoMin := string(ctx.PostArgs().Peek("deposit_time_two_min"))
	depositTimeTwoMax := string(ctx.PostArgs().Peek("deposit_time_two_max"))
	depositTimeThreeMin := string(ctx.PostArgs().Peek("deposit_time_three_min"))
	depositTimeThreeMax := string(ctx.PostArgs().Peek("deposit_time_three_max"))
	depositTimeOne := string(ctx.PostArgs().Peek("deposit_time_one"))
	depositTimeTwo := string(ctx.PostArgs().Peek("deposit_time_two"))
	depositTimeThree := string(ctx.PostArgs().Peek("deposit_time_three"))
	depositLevelLimit := string(ctx.PostArgs().Peek("deposit_level_limit"))
	depositListFirst := string(ctx.PostArgs().Peek("deposit_list_first"))
	depositThirdSwitch := string(ctx.PostArgs().Peek("deposit_third_switch"))
	depositCanRepeat := string(ctx.PostArgs().Peek("deposit_can_repeat"))
	code := string(ctx.PostArgs().Peek("code"))
	if depositTimeSwitch != "" {
		_, err := strconv.ParseFloat(depositTimeSwitch, 64)
		if err != nil {
			helper.Print(ctx, false, helper.ParamErr)
			return
		}
	}

	config := map[string]string{}
	config["deposit_time_switch"] = depositTimeSwitch
	config["deposit_time_one_min"] = depositTimeOneMin
	config["deposit_time_one_max"] = depositTimeOneMax
	config["deposit_time_two_min"] = depositTimeTwoMin
	config["deposit_time_two_max"] = depositTimeTwoMax
	config["deposit_time_three_min"] = depositTimeThreeMin
	config["deposit_time_three_max"] = depositTimeThreeMax
	config["deposit_time_one"] = depositTimeOne
	config["deposit_time_two"] = depositTimeTwo
	config["deposit_time_three"] = depositTimeThree
	config["deposit_level_limit"] = depositLevelLimit
	config["deposit_list_first"] = depositListFirst
	config["deposit_third_switch"] = depositThirdSwitch
	config["deposit_can_repeat"] = depositCanRepeat
	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	if !helper.CtypeDigit(code) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	err = model.ConfigUpdate(config)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

func (that *ConfigController) Withdraw(ctx *fasthttp.RequestCtx) {

	withdraw_usdt_min := string(ctx.PostArgs().Peek("withdraw_usdt_min"))
	withdraw_list_first := string(ctx.PostArgs().Peek("withdraw_list_first"))
	withdraw_auto_min := string(ctx.PostArgs().Peek("withdraw_auto_min"))
	withdraw_min := string(ctx.PostArgs().Peek("withdraw_min"))
	code := string(ctx.PostArgs().Peek("code"))
	if withdraw_usdt_min != "" {
		_, err := strconv.ParseFloat(withdraw_usdt_min, 64)
		if err != nil {
			helper.Print(ctx, false, helper.ParamErr)
			return
		}
	}

	config := map[string]string{}
	config["withdraw_usdt_min"] = withdraw_usdt_min
	config["withdraw_list_first"] = withdraw_list_first
	config["withdraw_auto_min"] = withdraw_auto_min
	config["withdraw_min"] = withdraw_min

	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	if !helper.CtypeDigit(code) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	err = model.ConfigUpdate(config)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

func (that *ConfigController) List(ctx *fasthttp.RequestCtx) {

	flag := string(ctx.PostArgs().Peek("flag"))
	username := string(ctx.PostArgs().Peek("username"))
	if !helper.CtypeDigit(flag) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}
	res, err := model.MemberConfigList(flag, username)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, res)
}

func (that *ConfigController) Insert(ctx *fasthttp.RequestCtx) {

	flag := string(ctx.PostArgs().Peek("flag"))
	username := string(ctx.PostArgs().Peek("username"))
	ty := string(ctx.PostArgs().Peek("ty"))
	if !helper.CtypeDigit(flag) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}
	err := model.MemberConfigInsert(flag, username, ty)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

func (that *ConfigController) Delete(ctx *fasthttp.RequestCtx) {

	id := string(ctx.PostArgs().Peek("id"))
	if !helper.CtypeDigit(id) {
		helper.Print(ctx, false, helper.IDErr)
		return
	}
	err := model.MemberConfigDelete(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}
