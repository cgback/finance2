package model

import (
	"database/sql"
	"time"
)

type TenantChannel struct {
	ID               string `db:"id" json:"id" cbor:"id"`
	TopID            string `db:"top_id" json:"top_id" cbor:"top_id"`                                     //总商户id
	TopName          string `db:"top_name" json:"top_name" cbor:"top_name"`                               //总商户名
	ParentID         string `db:"parent_id" json:"parent_id" cbor:"parent_id"`                            //上级商户id
	ParentName       string `db:"parent_name" json:"parent_name" cbor:"parent_name"`                      //上级商户名
	TenantID         string `db:"tenant_id" json:"tenant_id" cbor:"tenant_id"`                            // 商户id
	TenantName       string `db:"tenant_name" json:"tenant_name" cbor:"tenant_name"`                      //商户名
	Flags            string `db:"flags" json:"flags" cbor:"flags"`                                        //商户类型
	ChannelType      string `db:"channel_type" json:"channel_type" cbor:"channel_type"`                   //通道类型
	ThirdChannelID   string `db:"third_channel_id" json:"third_channel_id" cbor:"third_channel_id"`       //三方通道ID
	ThirdChannelName string `db:"third_channel_name" json:"third_channel_name" cbor:"third_channel_name"` //三方通道名
	MinAmount        string `db:"min_amount" json:"min_amount" cbor:"min_amount"`                         // 通道限额最小值
	MaxAmount        string `db:"max_amount" json:"max_amount" cbor:"max_amount"`                         // 通道限额，最大值
	State            string `db:"state" json:"state" cbor:"state"`                                        // 1启用 0禁用
}

// 结构体定义
type ChannelType struct {
	ID          string `db:"id" json:"id"`
	Name        string `db:"name" json:"name"`                                      // 通道类型名
	Alias       string `db:"alias" json:"alias"`                                    // 通道类型别名
	State       string `db:"state" json:"state"`                                    //状态 1启用 2禁用
	Sort        int    `db:"sort" json:"sort"`                                      //排序
	UpdatedAt   int64  `db:"updated_at" json:"updated_at" redis:"updated_at"`       //操作时间
	UpdatedUID  string `db:"updated_uid" json:"updated_uid" redis:"updated_uid"`    //操作人的ID
	UpdatedName string `db:"updated_name" json:"updated_name" redis:"updated_name"` //操作人的名字
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

// f_payment表名
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
	ID                string `db:"id" redis:"id" json:"id"`                               //id
	CateID            string `db:"cate_id" redis:"cate_id" json:"cate_id"`                //渠道ID
	ChannelID         string `db:"channel_id" redis:"channel_id" json:"channel_id"`       //支付方式id
	ChannelName       string `redis:"channel_name" json:"channel_name"`                   //通道id
	PaymentName       string `db:"payment_name" redis:"payment_name" json:"payment_name"` //通道名称
	Fmax              string `db:"fmax" redis:"fmax" json:"fmax"`                         //最大支付金额
	Fmin              string `db:"fmin" redis:"fmin" json:"fmin"`                         //最小支付金额
	AmountList        string `db:"amount_list" redis:"amount_list" json:"amount_list"`    // 固定金额列表
	Et                string `db:"et" redis:"et" json:"et"`                               //结束时间
	St                string `db:"st" redis:"st" json:"st"`                               //开始时间
	CreatedAt         string `db:"created_at" redis:"created_at" json:"created_at"`       //创建时间
	State             string `db:"state" redis:"state" json:"state"`                      //0:关闭1:开启
	Sort              string `db:"sort" redis:"sort" json:"sort"`                         //排序
	Comment           string `db:"comment" redis:"comment" json:"comment"`                //备注
	VipList           string `db:"vip_list" redis:"vip_list" json:"vip_list"`             //vip等级
	Discount          string `db:"discount" redis:"discount" json:"discount"`             //优惠
	UpdatedAt         string `db:"updated_at" redis:"updated_at" json:"updated_at"`       //更新时间
	Name              string `db:"name" redis:"name" json:"name"`                         //前端展示名称
	UpdatedName       string `db:"updated_name" redis:"updated_name" json:"updated_name"` //更新人
	IsZone            string `db:"is_zone" redis:"is_zone" json:"is_zone"`                //0没有1有区间
	IsFast            string `db:"is_fast" redis:"is_fast" json:"is_fast"`                //快捷金额是否开启
	Flag              string `db:"flag" redis:"flag" json:"flag"`                         //1三方通道2离线通道
	WebImg            string `db:"web_img" redis:"web_img" json:"web_img"`                //web端说明
	H5Img             string `db:"h5_img" redis:"h5_img" json:"h5_img"`                   //h5端说明
	AppImg            string `db:"app_img" redis:"app_img" json:"app_img"`                //App端说明
	DailyMaxAmount    string `db:"daily_max_amount" json:"daily_max_amount"`              // 当天最大收款限额
	DailyFinishAmount string `db:"daily_finish_amount" json:"daily_finish_amount"`        // 当天已收款总额
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

type Tunnel_t struct {
	ID         string `db:"id" json:"id"`                    //
	Name       string `db:"name" json:"name"`                //
	Sort       int    `db:"sort" json:"sort"`                //排序
	PromoState string `db:"promo_state"  json:"promo_state"` //存款优化开关
	//Content    string `db:"content"  json:"content"`         //存款优化开关
	PromoDiscount string `db:"promo_discount" json:"promo_discount"` // 存款优惠比例
	IsLastSuccess string `json:"is_last_success"`
}

// paymentDepositResp 存款
type paymentDepositResp struct {
	Addr       string                 // 三方返回的充值地址
	QrCode     string                 // 充值二维码地址
	PayCode    string                 // 附言信息
	Account    string                 // 收款卡号
	BankCode   string                 // 收款银行编码
	BankLogo   string                 // 收款银行Logo
	CardHolder string                 // 持卡人
	OrderID    string                 // 三方的订单号, 如果三方没有返回订单号, 这个值则为入参id(即我方订单号)
	Data       map[string]interface{} // 向三方发起http请求的参数以及response data
	IsForm     string
	UseLink    int //使用地址跳转或重新发起请求 0：使用链接跳转  1：使用订单号重新发起请求
}

type CallbackLog struct {
	OrderId      string `json:"order_id"`
	RequestURI   string `json:"requestURI"`
	RequestBody  string `json:"requestBody"`
	Error        string `json:"error"`
	ResponseBody string `json:"responseBody"`
	Index        string `json:"_index"`
}

type WithdrawAutoParam struct {
	OrderID     string    // 订单id
	Amount      string    // 金额
	BankID      string    // 银行id
	BankCode    string    // 银行
	CardNumber  string    // 银行卡号
	CardName    string    // 持卡人姓名
	Ts          time.Time // 时间
	PaymentID   string    // 提现渠道信息
	BankAddress string    // 开户支行
}

// paymentWithdrawalRsp 取款
type paymentWithdrawalRsp struct {
	OrderID string // 三方的订单号, 如果三方没有返回订单号, 这个值则为入参id(即我方订单号)
}

// 订单回调response
type paymentCallbackResp struct {
	OrderID string // 我方订单号
	State   int    // 订单状态
	Amount  string // 订单金额
	PayAt   string //支付时间时间戳(毫秒)
	Cent    int64  // 数据数值差异倍数
	Sign    string // 签名(g7的签名校验需要)
	Resp    interface{}
}

// FWithdrawData 取款数据
type FWithdrawData struct {
	T   int64      `json:"t"`
	D   []Withdraw `json:"d"`
	Agg Withdraw   `json:"agg"`
}

// Withdraw 会员提款表
type Withdraw struct {
	ID                string  `db:"id"                  json:"id"                 redis:"id"`
	Prefix            string  `db:"prefix"              json:"prefix"             redis:"prefix"`
	BID               string  `db:"bid"                 json:"bid"                redis:"bid"`                  //  下发的银行卡或虚拟钱包的ID
	Flag              int     `db:"flag"                json:"flag"               redis:"flag"`                 //  1=银行卡,2=虚拟钱包
	OID               string  `db:"oid"                 json:"oid"                redis:"oid"`                  //  三方ID
	UID               string  `db:"uid"                 json:"uid"                redis:"uid"`                  //
	ParentUID         string  `db:"parent_uid"          json:"parent_uid"         redis:"parent_uid"`           //  上级uid
	ParentName        string  `db:"parent_name"         json:"parent_name"        redis:"parent_name"`          // 上级代理名
	Username          string  `db:"username"            json:"username"           redis:"username"`             //
	PID               string  `db:"pid"                 json:"pid"                redis:"pid"`                  //  paymendID
	Amount            float64 `db:"amount"              json:"amount"             redis:"amount"`               // 提款金额
	State             int     `db:"state"               json:"state"              redis:"state"`                // 371:审核中 372:审核拒绝 373:出款中 374:提款成功 375:出款失败 376:异常订单 377:代付失败
	Automatic         int     `db:"automatic"           json:"automatic"          redis:"automatic"`            // 是否自动出款:0=手工,1=自动
	BankName          string  `db:"bank_name"           json:"bank_name"          redis:"bank_name"`            // 出款卡的银行名称
	RealName          string  `db:"real_name"           json:"real_name"          redis:"real_name"`            // 出款卡的开户人
	CardNo            string  `db:"card_no"             json:"card_no"            redis:"card_no"`              // 出款卡的卡号
	CreatedAt         int64   `db:"created_at"          json:"created_at"         redis:"created_at"`           //
	ConfirmAt         int64   `db:"confirm_at"          json:"confirm_at"         redis:"confirm_at"`           // 确认时间
	ConfirmUID        string  `db:"confirm_uid"         json:"confirm_uid"        redis:"confirm_uid"`          // 手动确认人uid
	ReviewRemark      string  `db:"review_remark"       json:"review_remark"      redis:"review_remark"`        // 审核备注
	WithdrawAt        int64   `db:"withdraw_at"         json:"withdraw_at"        redis:"withdraw_at"`          // 出款时间
	ConfirmName       string  `db:"confirm_name"        json:"confirm_name"       redis:"confirm_name"`         // 手动确认人名字
	WithdrawUID       string  `db:"withdraw_uid"        json:"withdraw_uid"       redis:"withdraw_uid"`         // 出款人的ID
	WithdrawName      string  `db:"withdraw_name"       json:"withdraw_name"      redis:"withdraw_name"`        //  出款人的名字
	WithdrawRemark    string  `db:"withdraw_remark"     json:"withdraw_remark"    redis:"withdraw_remark"`      // 出款备注
	FinanceType       int     `db:"finance_type"        json:"finance_type"       redis:"finance_type"`         // 财务类型 156=提款 165=代客提款 167=代理提款
	LastDepositAmount float64 `db:"last_deposit_amount" json:"last_deposit_amount" redis:"last_deposit_amount"` // 上笔成功存款金额
	RealNameHash      string  `db:"real_name_hash"      json:"real_name_hash"     redis:"real_name_hash"`       // 真实姓名哈希
	HangUpUID         string  `db:"hang_up_uid"         json:"hang_up_uid"        redis:"hang_up_uid"`          // 挂起人uid
	HangUpRemark      string  `db:"hang_up_remark"      json:"hang_up_remark"     redis:"hang_up_remark"`       // 挂起备注
	HangUpName        string  `db:"hang_up_name"        json:"hang_up_name"       redis:"hang_up_name"`         // 挂起人名字
	RemarkID          int64   `db:"remark_id"           json:"remark_id"          redis:"remark_id"`            // 挂起原因ID
	HangUpAt          int64   `db:"hang_up_at"          json:"hang_up_at"         redis:"hang_up_at"`           // 挂起时间
	ReceiveAt         int64   `db:"receive_at"          json:"receive_at"         redis:"receive_at"`           // 领取时间
	WalletFlag        int     `db:"wallet_flag"         json:"wallet_flag"        redis:"wallet_flag"`          // 钱包类型:1=中心钱包,2=佣金钱包
	TopUID            string  `db:"top_uid"             json:"top_uid"            redis:"top_uid"`              // 总代uid
	TopName           string  `db:"top_name"            json:"top_name"           redis:"top_name"`             // 总代用户名
	Level             int     `db:"level"               json:"level"              redis:"level"`
	Balance           string  `db:"balance"               json:"balance"              redis:"balance"`
	VirtualCount      float64 `db:"virtual_count" json:"virtual_count"` //数量
	VirtualRate       float64 `db:"virtual_rate" json:"virtual_rate"`   //汇率
	Currency          int     `db:"currency" json:"currency"`           //币种1=usdt
	Protocol          int     `db:"protocol" json:"protocol"`           //协议1=TRC20
	Alias             string  `db:"alias" json:"alias"`                 //别名
	WalletAddr        string  `db:"wallet_addr" json:"wallet_addr"`     //地址
}

type WithdrawListData struct {
	T   int64          `json:"t"`
	D   []withdrawCols `json:"d"`
	Agg Withdraw       `json:"agg"`
}

type withdrawCols struct {
	Withdraw
	CateID             string  `json:"cate_id"`
	CateName           string  `json:"cate_name"`
	MemberBankName     string  `json:"member_bank_name"`
	MemberBankNo       string  `json:"member_bank_no"`
	MemberBankRealName string  `json:"member_bank_real_name"`
	MemberBankAddress  string  `json:"member_bank_address"`
	MemberRealName     string  `json:"member_real_name"`
	MemberTags         string  `json:"member_tags"`
	Balance            string  `db:"balance"     json:"balance"     redis:"balance"    ` //余额
	LockAmount         float64 `db:"lock_amount" json:"lock_amount" redis:"lock_amount"` //锁定额度
	LastDepositAt      int     `json:"last_deposit_at"`
	FirstWithdraw      bool    `json:"first_withdraw"`
}

type MemberBankCard struct {
	ID           string `db:"id" json:"id"`
	UID          string `db:"uid" json:"uid"`
	Username     string `db:"username" json:"username"`
	BankAddress  string `db:"bank_address" json:"bank_address"`
	BankID       string `db:"bank_id" json:"bank_id"`
	BankBranch   string `db:"bank_branch_name" json:"bank_branch_name"`
	State        int    `db:"state" json:"state"`
	BankcardHash string `db:"bank_card_hash" json:"bank_card_hash"`
	CreatedAt    uint64 `db:"created_at" json:"created_at"`
}

type StateNum struct {
	T     int   `json:"t" db:"t"`
	State int64 `json:"state" db:"state"`
}

type MemberDepositInfo struct {
	Uid           string  `json:"uid" db:"uid"`
	DepositAmount float64 `json:"deposit_amount" db:"deposit_amount"`
	DepositAt     int     `json:"deposit_at" db:"deposit_at"`
	Prefix        string  `json:"prefix" db:"prefix"`
	Flags         int     `json:"flags" db:"flags"`
}

type paymentTDLog struct {
	Merchant     string `db:"merchant"`
	Channel      string `db:"channel"`
	Flag         string `db:"flag"`
	RequestURL   string `db:"request_url"`
	RequestBody  string `db:"request_body"`
	ResponseCode int    `db:"response_code"`
	ResponseBody string `db:"response_body"`
	Error        string `db:"error"`
	Lable        string `db:"lable"`
	Level        string `db:"level"`
	OrderID      string `db:"order_id"`
	Username     string `db:"username"`
}

type FConfig struct {
	Id      int64  `json:"id" db:"id"`
	Name    string `json:"name" db:"name"`
	Content string `json:"content" db:"content"`
	Prefix  string `json:"prefix" db:"prefix"`
}

type FMemberConfig struct {
	Id       int64  `db:"id" json:"id,omitempty"`
	Uid      string `db:"uid" json:"uid,omitempty"`
	Username string `db:"username" json:"username,omitempty"`
	Flag     string `db:"flag" json:"flag,omitempty"`
	Ty       string `db:"ty" json:"ty"`
}
