package ryrpc

import (
	"errors"
	"finance/contrib/helper"
	"fmt"
	"github.com/shopspring/decimal"
)

func CheckDepositFlow(username string) (bool, error) {

	ret := deposit_flow_check_resp_t{}
	err := client.Call("/promo/deposit/flow/check", username, &ret)
	if err != nil {
		return false, err
	}

	fmt.Println("CheckDepositFlow:", ret.Ok, ret.Amount)

	unFinishFlowStr := "0.00"
	if len(ret.Amount) > 0 {
		unFinishFlowDec, _ := decimal.NewFromString(ret.Amount)
		unFinishFlowStr = unFinishFlowDec.Truncate(2).StringFixed(2)
	}
	if !ret.Ok && ret.Amount != "0.00" {
		return false, errors.New(fmt.Sprintf(`Bạn Còn %sK Tiền Cược Chưa Hoàn Thành, Vui Lòng Hoàn Thành Cược Rồi Đăng Ký Lại.`, unFinishFlowStr))
	} else if !ret.Ok && ret.Amount == "0.00" {
		return false, errors.New(helper.RequestBusy)
	}

	return ret.Ok, nil
}
