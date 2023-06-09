package model

import (
	"database/sql"
	"time"
)

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

// Deposit 存款
type Deposit struct {
	ID              string  `db:"id" json:"id" redis:"id"`                                              //
	Prefix          string  `db:"prefix" json:"prefix" redis:"prefix"`                                  //转账后的金额
	OID             string  `db:"oid" json:"oid" redis:"oid"`                                           //转账前的金额
	UID             string  `db:"uid" json:"uid" redis:"uid"`                                           //用户ID
	Username        string  `db:"username" json:"username" redis:"username"`                            //用户名
	ChannelID       string  `db:"channel_id" json:"channel_id" redis:"channel_id"`                      //
	CID             string  `db:"cid" json:"cid" redis:"cid"`                                           //分类ID
	PID             string  `db:"pid" json:"pid" redis:"pid"`                                           //用户ID
	FinanceType     int     `db:"finance_type" json:"finance_type" redis:"finance_type"`                // 财务类型 441=充值 443=代客充值 445=代理充值
	Amount          float64 `db:"amount" json:"amount" redis:"amount"`                                  //金额
	USDTFinalAmount float64 `db:"usdt_final_amount" json:"usdt_final_amount" redis:"usdt_final_amount"` // 到账金额
	USDTApplyAmount float64 `db:"usdt_apply_amount" json:"usdt_apply_amount" redis:"usdt_apply_amount"` // 提单金额
	Rate            float64 `db:"rate" json:"rate" redis:"rate"`                                        // 汇率
	State           int     `db:"state" json:"state" redis:"state"`                                     //0:待确认:1存款成功2:已取消
	Automatic       int     `db:"automatic" json:"automatic" redis:"automatic"`                         //1:自动转账2:脚本确认3:人工确认
	CreatedAt       int64   `db:"created_at" json:"created_at" redis:"created_at"`                      //
	CreatedUID      string  `db:"created_uid" json:"created_uid" redis:"created_uid"`                   //创建人的ID
	CreatedName     string  `db:"created_name" json:"created_name" redis:"created_name"`                //创建人的名字
	ReviewRemark    string  `db:"review_remark" json:"review_remark" redis:"review_remark"`             //审核备注
	ConfirmAt       int64   `db:"confirm_at" json:"confirm_at" redis:"confirm_at"`                      //确认时间
	ConfirmUID      string  `db:"confirm_uid" json:"confirm_uid" redis:"confirm_uid"`                   //手动确认人id
	ConfirmName     string  `db:"confirm_name" json:"confirm_name" redis:"confirm_name"`                //手动确认人名字
	ProtocolType    string  `db:"protocol_type" json:"protocol_type" redis:"protocol_type"`             //地址类型 trc20 erc20
	Address         string  `db:"address" json:"address" redis:"address"`                               //收款地址
	HashID          string  `db:"hash_id" json:"hash_id" redis:"hash_id"`                               //区块链订单号
	Flag            int     `db:"flag" json:"flag" redis:"flag"`                                        // 1 三方订单 2 三方usdt订单 3 线下转卡订单 4 线下转usdt订单
	BankcardID      string  `db:"bankcard_id" json:"bankcard_id" redis:"bankcard_id"`                   // 线下转卡 收款银行卡id
	ManualRemark    string  `db:"manual_remark" json:"manual_remark" redis:"manual_remark"`             // 线下转卡订单附言
	BankCode        string  `db:"bank_code" json:"bank_code" redis:"bank_code"`                         // 银行编号
	BankNo          string  `db:"bank_no" json:"bank_no" redis:"bank_no"`                               // 银行卡号
	ParentUID       string  `db:"parent_uid" json:"parent_uid" redis:"parent_uid"`                      // 上级uid
	ParentName      string  `db:"parent_name" json:"parent_name" redis:"parent_name"`                   //上级代理名
	TopUID          string  `db:"top_uid" json:"top_uid" redis:"top_uid"`                               // 总代uid
	TopName         string  `db:"top_name" json:"top_name" redis:"top_name"`                            // 总代用户名
	Level           int     `db:"level" json:"level" redis:"level"`                                     //会员等级
	Discount        float64 `db:"discount" json:"discount" redis:"discount"`                            // 存款优惠/存款手续费
	GroupName       string  `db:"-" json:"group_name" redis:"group_name"`                               //团队名称
	SuccessTime     int     `db:"success_time" json:"success_time"`                                     //该用户第几笔成功的订单
}

// 存款数据
type FDepositData struct {
	T   int64             `json:"t"`
	D   []Deposit         `json:"d"`
	Agg map[string]string `json:"agg"`
}

type dataTotal struct {
	T sql.NullInt64   `json:"t"`
	S sql.NullFloat64 `json:"s"`
}
