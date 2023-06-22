package middleware

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/twmb/murmur3"
	"github.com/valyala/fasthttp"
	"github.com/xxtea/xxtea-go/xxtea"
	"net/url"
)

func checkHeaderMiddleware(ctx *fasthttp.RequestCtx) error {

	var (
		args   fasthttp.Args
		allows = map[string]bool{
			"/f2/callback/nvnd": true,
			"/f2/callback/nvnw": true,
		}
		forbidden = errors.New(`{"status":false,"data":"444"}`)
	)

	path := string(ctx.Path())
	fmt.Println(path)
	if _, ok := allows[path]; ok {
		return nil
	}

	device := string(ctx.Request.Header.Peek("d"))
	version := string(ctx.Request.Header.Peek("v"))
	reqTime := string(ctx.Request.Header.Peek("X-Ca-Timestamp"))
	nonce := string(ctx.Request.Header.Peek("X-Ca-Nonce"))

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
		//fmt.Println("device = ", device)
		//fmt.Println("keys = ")
		return forbidden
	}
	if version == "" {
		//fmt.Println("version = ")
		return forbidden
	}
	if reqTime == "" {
		//fmt.Println("reqTime = ")
		return forbidden
	}
	if nonce == "" {
		//fmt.Println("nonce = ")
		return forbidden
	}

	uri := string(ctx.RequestURI())
	decodedValue, err := url.QueryUnescape(uri)
	if err != nil {
		fmt.Println("url.QueryUnescape = ", err.Error())
		return forbidden
	}

	if ctx.IsGet() {
		str := fmt.Sprintf("%s%s%s%s", keys, reqTime, decodedValue, version)
		h2 := verify(str)
		if h2 != nonce {
			fmt.Println("GET h2 = ", h2)
			fmt.Println("GET nonce = ", nonce)
			fmt.Println("GET str = ", str)
			return forbidden
		}
	} else if ctx.IsPost() {

		b := string(ctx.PostBody())
		str := fmt.Sprintf("%s%s%s%s%s", b, keys, reqTime, decodedValue, version)
		h2 := verify(str)
		if h2 != nonce {
			fmt.Println("POST h2 = ", h2)
			fmt.Println("POST nonce = ", nonce)
			fmt.Println("POST str = ", str)
			return forbidden
		}
		data, err := base64.StdEncoding.DecodeString(b)
		if err != nil {
			fmt.Println("POST DecodeString err = ", err.Error())
			return forbidden
		}
		decryptData := xxtea.Decrypt(data, []byte(keys))

		//decryptData := string(xxtea.Decrypt(data, []byte(keys)))
		args.ParseBytes(decryptData)

		postArgs := ctx.PostArgs()
		postArgs.Reset()
		args.CopyTo(postArgs)
	} else {
		return forbidden
	}

	return nil
}

func verify(str string) string {

	h32 := murmur3.SeedNew32(24)
	h32.Write([]byte(str))
	v := h32.Sum32()

	return fmt.Sprintf("%d", v)
}
