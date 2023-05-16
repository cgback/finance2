package model

import "time"

// 常量定义
const (
	defaultRedisKeyPrefix = "rlock:"
)

var (
	LockTimeout = 20 * time.Second
)
