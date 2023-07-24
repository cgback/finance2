package middleware

import (
	"errors"
	"finance/model"
	"strconv"
	"strings"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"

	"finance/contrib/helper"
	"finance/contrib/session"
)

var allows = map[string]bool{
	"/f2/callback/nvnd": true,
	"/f2/callback/nvnw": true,
}

// 哪些路由不用动态密码验证
var otpIgnore = map[string]bool{}

func CheckTokenMiddleware(ctx *fasthttp.RequestCtx) error {

	path := string(ctx.Path())
	if _, ok := allows[path]; ok {
		return nil
	}

	var (
		data []byte
		err  error
	)
	if strings.HasPrefix(path, "/merchant") {

		//ip := helper.FromRequest(ctx)
		//if !model.WhiteListCheck(ip) {
		//	info := fmt.Sprintf("[%s] 不在白名单禁止访问", ip)
		//	//fmt.Println(info)
		//	return fmt.Errorf(`{"status":false,"data":"%s"}`, info)
		//}

		data, err = session.AdminGet(ctx)
		if err != nil {
			return errors.New(`{"status":false,"data":"token"}`)
		}

		_, ok := otpIgnore[path]
		if !ok && !otp(ctx, data) {
			// return errors.New(`{"status":false,"data":"otp"}`)
		}

		gid := fastjson.GetString(data, "group_id")
		permission := model.PrivCheck(path, gid)
		if permission != nil {
			return errors.New(`{"status":false,"data":"permission denied"}`)
		}
	} else {
		data, err = session.Get(ctx)
		if err != nil {
			return errors.New(`{"status":false,"data":"token"}`)
		}
	}

	ctx.SetUserValue("token", data)

	return nil
}

func otp(ctx *fasthttp.RequestCtx, data []byte) bool {

	seamo := ""
	if ctx.IsPost() {
		seamo = string(ctx.PostArgs().Peek("code"))
	} else if ctx.IsGet() {
		seamo = string(ctx.QueryArgs().Peek("code"))
	} else {
		return false
	}

	key := fastjson.GetString(data, "seamo")

	// fmt.Println("seamo= ", seamo)
	// fmt.Println("key= ", key)
	slat := helper.TOTP(key, 15)
	if s, err := strconv.Atoi(seamo); err != nil || s != slat {
		return false
	}
	// fmt.Println("seamo = ", key)

	return true
}
