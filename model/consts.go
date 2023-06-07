package model

import "time"

// 常量定义
const (
	defaultRedisKeyPrefix = "rlock:"
	QrTemplateQrOnly      = "qr_only"
)

var (
	LockTimeout = 20 * time.Second
)
