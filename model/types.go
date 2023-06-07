package model

import "time"

// 结构体定义
type ChannelType struct {
	ID    string `db:"id" json:"id"`
	Name  string `db:"name" json:"name"`   // 通道类型名
	Alias string `db:"alias" json:"alias"` // 通道类型别名
	State string `db:"state" json:"state"` //状态 1启用 2禁用
	Sort  int    `db:"sort" json:"sort"`   //排序
}

type TblBankTypes struct {
	Id        int    `json:"id" db:"id" redis:"-"`
	BankCode  string `json:"bank_code" db:"bank_code" redis:"-"`
	TrCode    string `json:"tr_code" db:"tr_code" redis:"tr_code"`
	NameCn    string `json:"name_cn" db:"name_cn" redis:"name_cn"`
	NameEn    string `json:"name_en" db:"name_en" redis:"name_en"`
	NameVn    string `json:"name_vn" db:"name_vn" redis:"name_vn"`
	ShortName string `json:"short_name" db:"short_name" redis:"short_name"`
	SwiftCode string `json:"swift_code" db:"swift_code" redis:"swift_code"`
	Alias     string `json:"alias" db:"alias" redis:"alias"`
	State     int    `json:"state" db:"state" redis:"state"`
	HasOtp    int    `json:"has_otp" db:"has_otp" redis:"has_otp"`
	Logo      string `json:"logo" db:"logo" redis:"logo"`
}

type DepositRsp struct {
	QrCode     string `json:"qr_code"`
	Amount     string `json:"amount"`
	Account    string `json:"account"`
	BankCode   string `json:"bank_code"`
	BankLogo   string `json:"bank_logo"`
	CardHolder string `json:"card_holder"`
	OrderNo    string `json:"order_no"`
	PayCode    string `json:"pay_code"`
}

type StbAdminLogs struct {
	Module    string    `bson:"module" json:"module"`
	Prefix    string    `bson:"prefix" json:"prefix"`
	Content   string    `bson:"content" json:"content"`
	Operation string    `bson:"operation" json:"operation"`
	AdminName string    `bson:"admin_name" json:"admin_name"`
	Ts        int64     `bson:"ts" json:"ts"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"` // ts 用于删除老记录
}
