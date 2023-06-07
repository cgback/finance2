package model

import (
	"finance/contrib/helper"
	"fmt"
	"time"
)

// 添加日志
func AdminLogInsert(module, content, operation, adminName string) error {

	ts := time.Now()
	record := StbAdminLogs{
		Prefix:    meta.Prefix,
		Module:    module,
		Content:   content,
		Operation: operation,
		AdminName: adminName,
		Ts:        ts.Unix(),
		CreatedAt: ts,
	}
	coll := meta.MgDB.Collection("stb_admin_logs")
	_, err := coll.InsertOne(ctx, record)
	if err != nil {
		fmt.Println("mongo insert error=", err)
		return pushLog(fmt.Errorf("%s,[%s]", err.Error(), "del mongo"), helper.DBErr)

	}
	return nil
}

const (
	InsertOp = "insert"
	UpdateOp = "update"
	DeleteOp = "delete"
	OpenOp   = "open"
	CloseOp  = "close"
	SetOp    = "set"
	PassOp   = "pass"
	RejectOp = "reject"
)

const (
	RiskModel     = "risk"
	FinanceModel  = "finance"
	DepositModel  = "deposit"  // 存款管理
	WithdrawModel = "withdraw" //提款管理
	ChannelModel  = "channel"  // 通道管理
)

const (
	STOP          = "停用"
	OPEN          = "启用"
	DEPOSIT       = "充值类型"
	WITHDRAW      = "提现类型"
	ThirdWithdraw = "三方代付"
	HandWithDraw  = "手动代付"
	PASS          = "通过"
	REJECT        = "拒绝"
)

var StateMap = map[string]string{
	"0": STOP,
	"1": OPEN,
}
var opMap = map[string]string{
	"0": CloseOp,
	"1": OpenOp,
}

var StateBoolMap = map[bool]string{
	false: STOP,
	true:  OPEN,
}
var opBoolMap = map[bool]string{
	false: CloseOp,
	true:  OpenOp,
}
var VipFlagMap = map[string]string{
	"1": DEPOSIT,
	"2": WITHDRAW,
}

var WithdrawTyMap = map[uint8]string{
	1: ThirdWithdraw,
	2: HandWithDraw,
}

var DepositReviewMap = map[string]string{
	"362": PASS,
	"363": REJECT,
}
var DepositOpMap = map[string]string{
	"362": PassOp,
	"363": RejectOp,
}
