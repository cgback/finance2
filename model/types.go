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

// FPay f_payment表名
type FPay struct {
	CateID    string `db:"cate_id" redis:"cate_id" json:"cate_id"`          //渠道ID
	ChannelID string `db:"channel_id" redis:"channel_id" json:"channel_id"` //通道id
	Comment   string `db:"comment" redis:"comment" json:"comment"`          //
	CreatedAt string `db:"created_at" redis:"created_at" json:"created_at"` //创建时间
	Et        string `db:"et" redis:"et" json:"et"`                         //结束时间
	Fmax      string `db:"fmax" redis:"fmax" json:"fmax"`                   //最大支付金额
	Fmin      string `db:"fmin" redis:"fmin" json:"fmin"`                   //最小支付金额
	Gateway   string `db:"gateway" redis:"gateway" json:"gateway"`          //支付网关
	ID        string `db:"id" redis:"id" json:"id"`                         //
	Quota     string `db:"quota" redis:"quota" json:"quota"`                //每天限额
	Amount    string `db:"amount" redis:"amount" json:"amount"`             //每天限额
	Sort      string `db:"sort" redis:"sort" json:"sort"`                   //
	St        string `db:"st" redis:"st" json:"st"`                         //开始时间
	State     string `db:"state" redis:"state" json:"state"`                //0:关闭1:开启
	Devices   string `db:"devices" redis:"devices" json:"devices"`          //设备号
}

type Payment_t struct {
	CateID      string `db:"cate_id" redis:"cate_id" json:"cate_id"`                //渠道ID
	ChannelID   string `db:"channel_id" redis:"channel_id" json:"channel_id"`       //通道id
	ChannelName string `redis:"channel_name" json:"channel_name"`                   //通道id
	PaymentName string `db:"payment_name" redis:"payment_name" json:"payment_name"` //子通道名称
	Comment     string `db:"comment" redis:"comment" json:"comment"`                //
	CreatedAt   string `db:"created_at" redis:"created_at" json:"created_at"`       //创建时间
	Et          string `db:"et" redis:"et" json:"et"`                               //结束时间
	Fmax        string `db:"fmax" redis:"fmax" json:"fmax"`                         //最大支付金额
	Fmin        string `db:"fmin" redis:"fmin" json:"fmin"`                         //最小支付金额
	Gateway     string `db:"gateway" redis:"gateway" json:"gateway"`                //支付网关
	ID          string `db:"id" redis:"id" json:"id"`                               //
	Quota       string `db:"quota" redis:"quota" json:"quota"`                      //每天限额
	Amount      string `db:"amount" redis:"amount" json:"amount"`                   //每天限额
	Sort        string `db:"sort" redis:"sort" json:"sort"`                         //
	St          string `db:"st" redis:"st" json:"st"`                               //开始时间
	State       string `db:"state" redis:"state" json:"state"`                      //0:关闭1:开启
	Devices     string `db:"devices" redis:"devices" json:"devices"`                //设备号
	AmountList  string `db:"amount_list" redis:"amount_list" json:"amount_list"`    // 固定金额列表
}
