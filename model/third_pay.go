package model

import (
	"errors"
	"finance/contrib/helper"
	"fmt"
	"github.com/valyala/fasthttp"
	"net/url"
	"time"
)

const (
	QRBanking  = "1"
	MomoPay    = "2"
	ZaloPay    = "3"
	ViettelPay = "4"
	TheCao     = "5"
	bankPay    = "101"
)

type nvnPayConf struct {
	Name           string
	AppID          string
	Key            string
	Api            string
	QrType         string
	PayNotify      string
	WithdrawNotify string
	Channel        map[string]string
}

type NvnPayment struct {
	Conf nvnPayConf
}

type nvnPayResp struct {
	Status bool `json:"status"`
	Data   struct {
		QrCode      string `json:"qr_code"`
		Amount      string `json:"amount"`
		Account     string `json:"account"`
		BankCode    string `json:"bank_code"`
		BankLogo    string `json:"bank_logo"`
		CardHolder  string `json:"card_holder"`
		OrderNo     string `json:"order_no"`
		PayCode     string `json:"pay_code"`
		IsRedirect  bool   `json:"is_redirect"`
		RedirectURL string `json:"redirect_url"`
	} `json:"data"`
}

type nvnPayWithdrawResp struct {
	Status bool `json:"status"`
	Data   struct {
		BillNo string `json:"bill_no"`
		State  int    `json:"state"`
	} `json:"data"`
}

type nvnPayCallBack struct {
	Status bool   `json:"status"`
	Data   string `json:"data"`
}

func (that *NvnPayment) New() {

	that.Conf = nvnPayConf{
		AppID:          meta.Finance["nvn"]["app_id"].(string),
		Key:            meta.Finance["nvn"]["key"].(string),
		Name:           "vnPay",
		Api:            meta.Finance["nvn"]["api"].(string),
		QrType:         meta.Finance["nvn"]["qr_type"].(string),
		PayNotify:      "%s/finance/callback/nvnd",
		WithdrawNotify: "%s/finance/callback/nvnw",
		Channel: map[string]string{
			"1": QRBanking,
			"2": MomoPay,
			"3": ZaloPay,
			"4": ViettelPay,
			"5": TheCao,
		},
	}
}

func (that *NvnPayment) Name() string {
	return that.Conf.Name
}

func (that *NvnPayment) Pay(orderId, ch, amount, bid string) (paymentDepositResp, error) {

	data := paymentDepositResp{}

	amount = fmt.Sprintf("%s000", amount)
	id := that.Conf.AppID
	callbackUrl := fmt.Sprintf(that.Conf.PayNotify, meta.FcallbackInner)

	flags := that.Conf.Channel[ch]
	params := url.Values{}
	params.Set("amount", amount)
	params.Set("flags", flags)
	params.Set("id", id)
	params.Set("order_no", orderId)
	params.Set("callback_url", callbackUrl)
	params.Set("qr_type", that.Conf.QrType)
	params.Set("ts", fmt.Sprintf("%d", time.Now().Unix()))
	params.Set("is_direct", fmt.Sprintf("%d", meta.IsDirect))
	signArgs := fmt.Sprintf("%s|%s|%s|%s|%s|%s", amount, flags, id, orderId, callbackUrl, that.Conf.Key)
	sign := helper.MD5Hash(signArgs)
	params.Set("sign", sign)

	postBody := params.Encode()
	header := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	uri := fmt.Sprintf("%s/api/deposit/create", that.Conf.Api)
	body, err := httpDoTimeout("nvnPay", []byte(postBody), "POST", uri, header, time.Second*12)
	//fmt.Println(uri, postBody, sign, string(body))
	ts := time.Now()
	lg := CallbackLog{
		OrderId:      orderId,
		RequestURI:   uri,
		RequestBody:  postBody,
		Error:        "nil",
		ResponseBody: "nil",
		Index:        fmt.Sprintf("%s_cgpay_deposit%04d%02d", meta.Prefix, ts.Year(), ts.Month()),
	}
	if err != nil {
		lg.Error = err.Error()
	}
	defer func() {
		lg.ResponseBody = string(body)
		payload, _ := helper.JsonMarshal(lg)
		_ = RocketSendAsync("zinc_fluent_log", payload)
	}()
	if err != nil {
		return data, errors.New(helper.PayServerErr)
	}

	rsp := nvnPayResp{}
	if err = helper.JsonUnmarshal(body, &rsp); err != nil {
		_ = pushLog(fmt.Errorf("json format err: %s", err.Error()), helper.FormatErr)
		return data, fmt.Errorf(helper.ServerErr)
	}

	data.QrCode = rsp.Data.QrCode
	data.PayCode = rsp.Data.PayCode
	data.OrderID = rsp.Data.OrderNo
	data.Account = rsp.Data.Account
	data.CardHolder = rsp.Data.CardHolder
	data.BankCode = rsp.Data.BankCode
	data.BankLogo = rsp.Data.BankLogo
	data.UseLink = 1
	// 刮刮卡直接跳转
	if rsp.Data.IsRedirect {
		data.UseLink = 0
		data.Addr = rsp.Data.RedirectURL
	}

	rsp.Data.QrCode = "qrcode:image/png;base64"
	body, _ = helper.JsonMarshal(rsp)
	return data, nil
}

func (that *NvnPayment) Withdraw(arg WithdrawAutoParam) (paymentWithdrawalRsp, error) {
	data := paymentWithdrawalRsp{}

	id := that.Conf.AppID
	callbackUrl := fmt.Sprintf(that.Conf.WithdrawNotify, meta.FcallbackInner)

	params := url.Values{}
	params.Set("amount", arg.Amount)
	params.Set("flags", bankPay)
	params.Set("id", id)
	params.Set("order_no", arg.OrderID)
	params.Set("callback_url", callbackUrl)
	params.Set("payee_bank_code", arg.BankCode)
	params.Set("payee_account", arg.CardNumber)
	params.Set("payee_name", arg.CardName)
	params.Set("ts", fmt.Sprintf("%d", time.Now().Unix()))
	signArgs := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s",
		arg.BankCode, arg.CardNumber, arg.CardName, arg.Amount, bankPay, id, arg.OrderID, callbackUrl, that.Conf.Key)
	sign := helper.MD5Hash(signArgs)
	fmt.Println("check sign: ", signArgs, sign)
	params.Set("sign", sign)

	postBody := params.Encode()
	header := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	uri := fmt.Sprintf("%s/api/withdraw/create", that.Conf.Api)
	body, err := httpDoTimeout("nvnPay", []byte(postBody), "POST", uri, header, time.Second*12)
	//fmt.Println(uri, postBody, sign, string(body))
	ts := time.Now()
	lg := CallbackLog{
		OrderId:      arg.OrderID,
		RequestURI:   uri,
		RequestBody:  postBody,
		Error:        "nil",
		ResponseBody: string(body),
		Index:        fmt.Sprintf("%s_cgpay_withdraw%04d%02d", meta.Prefix, ts.Year(), ts.Month()),
	}
	if err != nil {
		lg.Error = err.Error()
	}
	payload, _ := helper.JsonMarshal(lg)
	_ = RocketSendAsync("zinc_fluent_log", payload)
	if err != nil {
		return data, errors.New(helper.PayServerErr)
	}

	rsp := nvnPayWithdrawResp{}
	if err = helper.JsonUnmarshal(body, &rsp); err != nil {
		_ = pushLog(fmt.Errorf("json format err: %s", err.Error()), helper.FormatErr)
		return data, fmt.Errorf(helper.ServerErr)
	}

	data.OrderID = rsp.Data.BillNo

	return data, nil
}

func (that *NvnPayment) PayCallBack(fCtx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	tenantID := string(fCtx.PostArgs().Peek("tenant_id"))
	orderNo := string(fCtx.PostArgs().Peek("order_no"))
	amount := string(fCtx.PostArgs().Peek("amount"))
	state := string(fCtx.PostArgs().Peek("state"))
	payAt := string(fCtx.PostArgs().Peek("pay_at"))
	sign := string(fCtx.PostArgs().Peek("sign"))

	rsp := paymentCallbackResp{
		State: DepositConfirming,
		PayAt: payAt,
	}

	if tenantID != that.Conf.AppID {
		return rsp, fmt.Errorf("invalid tenant")
	}
	signArgs := fmt.Sprintf("%s|%s|%s|%s|%s", tenantID, orderNo, amount, state, that.Conf.Key)
	if sign != helper.MD5Hash(signArgs) {
		return rsp, fmt.Errorf("invalid sign")
	}

	if state == "2" {
		rsp.State = DepositSuccess
	}

	rsp.OrderID = orderNo
	rsp.Amount = amount
	rsp.Resp = &nvnPayCallBack{
		Status: true,
		Data:   "Success",
	}

	return rsp, nil
}

func (that *NvnPayment) WithdrawCallBack(fCtx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	tenantID := string(fCtx.PostArgs().Peek("tenant_id"))
	orderNo := string(fCtx.PostArgs().Peek("order_no"))
	amount := string(fCtx.PostArgs().Peek("amount"))
	state := string(fCtx.PostArgs().Peek("state"))
	payAt := string(fCtx.PostArgs().Peek("pay_at"))
	sign := string(fCtx.PostArgs().Peek("sign"))
	rsp := paymentCallbackResp{
		PayAt: payAt,
	}

	if tenantID != that.Conf.AppID {
		return rsp, fmt.Errorf("invalid tenant")
	}

	signArgs := fmt.Sprintf("%s|%s|%s|%s|%s", tenantID, orderNo, amount, state, that.Conf.Key)
	if sign != helper.MD5Hash(signArgs) {
		return rsp, fmt.Errorf("invalid sign")
	}

	if state == "2" {
		rsp.State = WithdrawSuccess
	} else if state == "3" {
		rsp.State = WithdrawAutoPayFailed
	}

	rsp.OrderID = orderNo
	rsp.Amount = amount
	rsp.Resp = nvnPayCallBack{
		Status: true,
		Data:   "Success",
	}

	return rsp, nil
}
