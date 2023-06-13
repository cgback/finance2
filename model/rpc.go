package model

import (
	"context"
	"errors"
	"finance/contrib/helper"
	"fmt"
	"github.com/hprose/hprose-golang/v3/rpc/core"
	"github.com/shopspring/decimal"
	"net/http"
)

var grpc_t struct {
	View       func(rctx context.Context, uid, field string) ([]string, error)
	Encrypt    func(rctx context.Context, uid string, data [][]string) error
	Decrypt    func(rctx context.Context, uid string, hide bool, field []string) (map[string]string, error)
	DecryptAll func(rctx context.Context, uids []string, hide bool, field []string) (map[string]map[string]string, error)

	CheckDepositFlow  func(rctx context.Context, username string) (bool, string)
	FinishDepositFlow func(rctx context.Context, username, billNo, adminId, adminName string) bool
}

func rpcCheckFlow(username string) (bool, error) {

	// 检查上次提现成功到现在的存款流水是否满足 未满足的返回流水未达标
	clientContext := core.NewClientContext()
	header := make(http.Header)
	header.Set("X-Func-Name", "CheckDepositFlow")
	clientContext.Items().Set("httpRequestHeaders", header)
	rctx := core.WithContext(context.Background(), clientContext)

	ok, unFinishFlow := grpc_t.CheckDepositFlow(rctx, username)
	fmt.Println("CheckDepositFlow:", ok, unFinishFlow)
	unFinishFlowStr := "0.00"
	if len(unFinishFlow) > 0 {
		unFinishFlowDec, _ := decimal.NewFromString(unFinishFlow)
		unFinishFlowStr = unFinishFlowDec.Truncate(2).StringFixed(2)
	}
	if !ok && unFinishFlow != "0.00" {
		return false, errors.New(fmt.Sprintf(`Bạn Còn %sK Tiền Cược Chưa Hoàn Thành, Vui Lòng Hoàn Thành Cược Rồi Đăng Ký Lại.`, unFinishFlowStr))
	} else if !ok && unFinishFlow == "0.00" {
		return false, errors.New(helper.RequestBusy)
	}

	return ok, nil
}

// merchantID nvn支付配置中的app_id
func rpcDepositChannelList(merchantID string) ([]TenantChannel, error) {

	var data []TenantChannel
	err := meta.MerchantRPC.Call("/pay/deposit/channel/list", merchantID, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}
