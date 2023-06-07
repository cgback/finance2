package model

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"time"
)

var (
	bankCodeMap = map[string]string{
		"1005": "970423", //TPBANK
		"1006": "970405", //Agribank
		"1007": "970414", //Oceanbank
		"1008": "970418", //BIDV
		"1009": "970415", //VietinBank
		"1010": "970436", //Vietcombank
		"1011": "970432", //VPBANK
		"1012": "970422", //MBBank
		"1013": "970407", //Techcombank
		"1014": "970403", //Sacombank
		"1015": "970441", //VIB
		"1016": "970426", //MSB
		"1017": "970431", //Eximbank
		"1018": "970409", //BAC A BANK
		"1019": "970400", //SAIGONBANK
		"1020": "970424", //Shinhan Bank
		"1021": "458761", //HSBC
		"1022": "970457", //Woori
		"1023": "970421", //VRB
		"1024": "970458", //United Overseas
		"1025": "970439", //PBVN
		"1026": "970430", //PG Bank
		"1027": "970408", //GPBANK
		"1028": "970434", //IVB
		"1029": "970442", //HLBANK
		"1030": "970452", //KienlongBank
		"1031": "970428", //NamABank
		"1032": "970419", //NCB
		"1033": "970448", //OCB
		"1034": "970427", //VietABank
		"1035": "970438", //BaoViet
		"1036": "970433", //VIETBANK
		"1037": "422589", //CIMB
		"1038": "970449", //LienVietPostBank
		"1039": "970425", //ABBANK
		"1040": "970454", //VIET CAPITAL BANK
		"1041": "970437", //HDBank
		"1042": "970429", //SCB
		"1043": "970406", //DONG A BANK
		"1044": "970443", //SHB
		"1045": "970440", //SeABank
		"1046": "970412", //PVcomBank
		"1047": "1047",   //VBSP
		"1048": "1048",   //VDB
		"1049": "970410", //Chartered
		"1050": "970444", //CBB
		"1051": "970416", //ACB
		"1052": "796500", //DBSBank
		"1053": "970455", //IBK-HN
		"1054": "970456", //IBK-HCM
		"1055": "801011", //NONGHUYP
		"1056": "970462", //KookminHN
		"1057": "970463", //KookminHCM
		"1058": "970446", //COOPBANK
		"1059": "546034", //CAKE
		"1060": "546035", //UBANK
		"1061": "668888", //KBank
	}
)

func qrVietQr(bankCode, account, template, amount, addInfo, accountName string) (string, error) {

	uri := fmt.Sprintf("https://img.vietqr.io/image/%s-%s-%s.png?amount=%s&addInfo=%s&accountName=%s",
		bankCode, account, template, url.QueryEscape(amount), url.QueryEscape(addInfo), url.QueryEscape(accountName))
	fmt.Println(uri)
	body, err := httpDoTimeout("offline", []byte{}, "GET", uri, nil, 8*time.Second)
	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}

	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(body), nil
}

func QrCodeGen(b Bankcard_t, amount, code string) (string, error) {

	rsp := DepositRsp{}
	fmt.Println(b.ChannelBankId)
	rsp.BankCode = bankCodeMap[b.ChannelBankId]
	rsp.Account = b.BanklcardNo
	rsp.CardHolder = b.AccountName
	rsp.Amount = amount
	rsp.PayCode = code

	qrKey := fmt.Sprintf("%s:offline:qr:%s:%s:%s", meta.Prefix, rsp.BankCode, rsp.Account, amount)
	qr, err := qrVietQr(rsp.BankCode, rsp.Account, QrTemplateQrOnly, amount, rsp.PayCode, rsp.CardHolder)
	if err != nil {
		//实时获取二维码不成功，取缓存中不带附言的二维码
		qr, err = meta.MerchantRedis.Get(ctx, qrKey).Result()
		if err == nil {
			rsp.QrCode = qr
			return rsp.QrCode, nil
		}

		return rsp.QrCode, err
	}

	rsp.QrCode = qr

	// 没有该银行卡，对应金额的收款二维码缓存，获取不带附言的二维码，缓存
	if meta.MerchantRedis.Exists(ctx, qrKey).Val() == 0 {
		// 不带附言二维码
		qr, err = qrVietQr(rsp.BankCode, rsp.Account, QrTemplateQrOnly, amount, "", rsp.CardHolder)
		if err == nil {
			pipe := meta.MerchantRedis.Pipeline()
			pipe.Set(ctx, qrKey, qr, 100*time.Hour)
			pipe.Persist(ctx, qrKey)
			_, _ = pipe.Exec(ctx)
			_ = pipe.Close()
		} else {
			//没有就拿后台默认的 不带金额的缓存
			//TODO
		}
	}

	return rsp.QrCode, nil
}
