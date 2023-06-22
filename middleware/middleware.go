package middleware

import (
	b64 "encoding/base64"
	"finance/model"
	"fmt"
	"github.com/valyala/fasthttp"
	"github.com/xxtea/xxtea-go/xxtea"
	"time"
)

var (
	isDev           = false
	validateH5      = ""
	validateHT      = ""
	validateWEB     = ""
	validateAndroid = ""
	validateIOS     = ""
)

type middleware_t func(ctx *fasthttp.RequestCtx) error

var MiddlewareList = []middleware_t{
	checkHeaderMiddleware,
	CheckTokenMiddleware,
}

func Use(next fasthttp.RequestHandler, prefix, h5, ht, web, android, ios string, dev bool) fasthttp.RequestHandler {

	validateH5 = h5
	validateHT = ht
	validateWEB = web
	validateAndroid = android
	validateIOS = ios
	isDev = dev

	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {

		for _, cb := range MiddlewareList {
			if err := cb(ctx); err != nil {
				fmt.Println("validateHT:", validateHT)
				fmt.Println("isDev:", isDev)
				if isDev {
					fmt.Fprint(ctx, err)
					return
				}

				device := string(ctx.Request.Header.Peek("d"))
				keys := ""
				switch device {
				case "23":
					keys = validateHT
				case "24":
					keys = validateWEB
				case "25":
					keys = validateH5
				case "35":
					keys = validateAndroid
				case "36":
					keys = validateIOS
				}
				if keys == "" {
					fmt.Fprint(ctx, err)
					return
				}

				encryptData := xxtea.Encrypt([]byte(err.Error()), []byte(keys))
				sEnc := b64.StdEncoding.EncodeToString(encryptData)
				ctx.SetContentType("text/plain")
				ctx.SetBody([]byte(sEnc))

				return
			}
		}

		st := time.Now()
		next(ctx)
		cost := time.Since(st)
		if cost > 2*time.Second {
			path := string(ctx.Path())
			info := fmt.Sprintf("path: %s:%s, query args: %s, post args: %s, d: %s, r: %s, ts: %s, time cost: %v",
				prefix, path, ctx.QueryArgs().String(), ctx.PostArgs().String(),
				string(ctx.Request.Header.Peek("d")), string(ctx.QueryArgs().Peek("r")),
				st.Format("2006-01-02 15:04:05"), cost)
			fmt.Println(info)
			_ = model.RocketSendAsync("telegram_bot_alert", []byte(info))
		}
	})
}
