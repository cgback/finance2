package model

// 结构体定义
type ChannelType struct {
	ID    string `db:"id" json:"id"`
	Name  string `db:"name" json:"name"`   // 通道类型名
	Alias string `db:"alias" json:"alias"` // 通道类型别名
	State string `db:"state" json:"state"` //状态 1启用 2禁用
	Sort  int    `db:"sort" json:"sort"`   //排序
}
