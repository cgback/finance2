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
	LockTimeout   = 20 * time.Second
	paymentLogTag = "payment_log"
	// 通过redis锁定提款订单的key
	depositOrderLockKey = "d:order:%s"
	// 通过redis锁定提款订单的key
	withdrawOrderLockKey = "w:order:%s"
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

var StateDesMap = map[int]string{
	DepositSuccess:   "通过",
	DepositCancelled: "拒绝",
}
var OpDesMap = map[int]string{
	DepositSuccess:   PassOp,
	DepositCancelled: RejectOp,
}

var defaultLevelWithdrawLimit = map[string]string{
	"count_remain":   "7",
	"max_remain":     "700000",
	"withdraw_count": "7",
	"withdraw_max":   "700000",
}

// 后台上下分审核状态
const (
	AdjustReviewing    = 256 //后台调整审核中
	AdjustReviewPass   = 257 //后台调整审核通过
	AdjustReviewReject = 258 //后台调整审核不通过
)

// 后台上下分状态
const (
	AdjustFailed      = 261 //上下分失败
	AdjustSuccess     = 262 //上下分成功
	AdjustPlatDealing = 263 //上分场馆处理中
)
