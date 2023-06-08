package model

import "time"

// 常量定义
const (
	defaultRedisKeyPrefix = "rlock:"
	QrTemplateQrOnly      = "qr_only"
	DepositFlagThird      = 1
	DepositFlagThirdUSTD  = 2
	DepositFlagManual     = 3
	DepositFlagUSDT       = 4
)

var (
	LockTimeout = 20 * time.Second
)

// 取款状态
const (
	WithdrawReviewing     = 371 //审核中
	WithdrawReviewReject  = 372 //审核拒绝
	WithdrawDealing       = 373 //出款中
	WithdrawSuccess       = 374 //提款成功
	WithdrawFailed        = 375 //出款失败
	WithdrawAbnormal      = 376 //异常订单
	WithdrawAutoPayFailed = 377 // 代付失败
	WithdrawHangup        = 378 // 挂起
	WithdrawDispatched    = 379 // 已派单
)

// 存款状态
const (
	DepositConfirming = 361 //确认中
	DepositSuccess    = 362 //存款成功
	DepositCancelled  = 363 //存款已取消
	DepositReviewing  = 364 //存款审核中
)
