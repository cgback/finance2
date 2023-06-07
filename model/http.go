package model

import (
	"fmt"
	"github.com/valyala/fasthttp"
	"time"
)

func httpDoTimeout(merchant string, requestBody []byte, method string, requestURI string, headers map[string]string, timeout time.Duration) ([]byte, error) {

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	//fmt.Println("****")
	fmt.Println("requestURI = ", requestURI)
	fmt.Println("requestBody = ", string(requestBody))
	defer func() {
		fasthttp.ReleaseResponse(resp)
		fasthttp.ReleaseRequest(req)
	}()

	req.SetRequestURI(requestURI)
	req.Header.SetMethod(method)

	switch method {
	case "POST":
		req.SetBody(requestBody)
	}

	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	err := fc.DoTimeout(req, resp, timeout)

	code := resp.StatusCode()
	respBody := resp.Body()

	if err != nil {
		return respBody, fmt.Errorf("send http request error: [%v]", err)
	}

	if code != fasthttp.StatusOK {
		return respBody, fmt.Errorf("bad http response code: [%d]", code)
	}

	return respBody, nil
}
