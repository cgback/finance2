package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	ryrpc "finance/rpc"
	"fmt"
	"github.com/tenfyzhong/cityhash"
	"strconv"
	"time"

	g "github.com/doug-martin/goqu/v9"
	"github.com/go-redis/redis/v8"
	"github.com/shopspring/decimal"
	"github.com/valyala/fasthttp"
)

type VirtualWallet_t struct {
	Id          string  `db:"id" json:"id" json:"id"`
	Name        string  `db:"name" json:"name"`
	Currency    string  `db:"currency" json:"currency"`                              // 1 usdt
	Pid         string  `db:"pid" json:"pid"`                                        // f_payment 的 id
	Protocol    string  `db:"protocol" json:"protocol"`                              // 协议
	WalletAddr  string  `db:"wallet_addr" json:"wallet_addr"`                        // 钱包地址
	State       string  `db:"state" json:"state"`                                    // 状态：2 关闭  1 开启
	MaxAmount   float64 `db:"max_amount" json:"max_amount"`                          // 充值最大金额
	MinAmount   float64 `db:"min_amount" json:"min_amount"`                          // 充值最小金额
	QrImg       string  `db:"qr_img" json:"qr_img"`                                  //logo
	Sort        int     `db:"sort" json:"sort"`                                      //排序
	Remark      string  `db:"remark" json:"remark"`                                  //备注
	CreatedAt   int64   `db:"created_at" json:"created_at" redis:"created_at"`       //创建时间
	CreatedUID  string  `db:"created_uid" json:"created_uid" redis:"created_uid"`    //创建人的ID
	CreatedName string  `db:"created_name" json:"created_name" redis:"created_name"` //创建人的名字
	UpdatedAt   int64   `db:"updated_at" json:"updated_at" redis:"updated_at"`       //操作时间
	UpdatedUID  string  `db:"updated_uid" json:"updated_uid" redis:"updated_uid"`    //操作人的ID
	UpdatedName string  `db:"updated_name" json:"updated_name" redis:"updated_name"` //操作人的名字
}

type VirtualWalletData struct {
	D []VirtualWallet_t `json:"d"`
	T int64             `json:"t"`
	S uint16            `json:"s"`
}

type MemberVirtualWallet struct {
	ID          string `db:"id" json:"id"`
	UID         string `db:"uid" json:"uid"`
	Username    string `db:"username" json:"username"`
	WalletAddr  string `db:"wallet_addr" json:"wallet_addr"`
	Currency    string `db:"currency" json:"currency"`
	Protocol    string `db:"protocol" json:"protocol"`
	State       int    `db:"state" json:"state"`
	Alias       string `db:"alias" json:"alias"`
	CreatedAt   int64  `db:"created_at" json:"created_at"`
	UpdatedAt   int64  `db:"updated_at" json:"updated_at"`
	UpdatedName string `db:"updated_name" json:"updated_name"`
	UpdatedUid  string `db:"updated_uid" json:"updated_uid"`
}

func UsdtUpdate(depositUsdtRate, withdrawUsdtRate, adminName string) error {

	if depositUsdtRate != "" {
		query := fmt.Sprintf(`insert into f_config(name, content,prefix) values ('%s', '%s','%s') on duplicate key update name = '%s', content = '%s',prefix= '%s'`, "deposit_usdt_rate", depositUsdtRate, meta.Prefix, "deposit_usdt_rate", depositUsdtRate, meta.Prefix)
		_, err := meta.MerchantDB.Exec(query)
		if err != nil {
			return errors.New(helper.DBErr)
		}
		err = meta.MerchantRedis.HSet(ctx, meta.Prefix+":usdt", "deposit_usdt_rate", depositUsdtRate).Err()
		if err != nil {
			return errors.New(helper.RedisErr)
		}
		contentLog := fmt.Sprintf("渠道管理-USDT汇率管理-修改:后台账号:%s【%s为:%s", adminName, "充值汇率", depositUsdtRate)
		AdminLogInsert(ChannelModel, contentLog, SetOp, adminName)
	}

	if withdrawUsdtRate != "" {
		query := fmt.Sprintf(`insert into f_config(name, content,prefix) values ('%s', '%s','%s') on duplicate key update name = '%s', content = '%s',prefix= '%s'`, "withdraw_usdt_rate", withdrawUsdtRate, meta.Prefix, "withdraw_usdt_rate", withdrawUsdtRate, meta.Prefix)
		_, err := meta.MerchantDB.Exec(query)
		if err != nil {
			return errors.New(helper.DBErr)
		}
		err = meta.MerchantRedis.HSet(ctx, meta.Prefix+":usdt", "withdraw_usdt_rate", withdrawUsdtRate).Err()
		if err != nil {
			return errors.New(helper.RedisErr)
		}
		contentLog := fmt.Sprintf("渠道管理-USDT汇率管理-修改:后台账号:%s【%s为:%s", adminName, "提现汇率", withdrawUsdtRate)
		AdminLogInsert(ChannelModel, contentLog, SetOp, adminName)
	}

	return nil
}

func UsdtInfo() (map[string]string, error) {

	res := map[string]string{}
	f, err := meta.MerchantRedis.HMGet(ctx, meta.Prefix+":usdt", "deposit_usdt_rate", "withdraw_usdt_rate").Result()
	if err != nil && redis.Nil != err {
		return res, errors.New(helper.RedisErr)
	}

	drate := ""
	wrate := ""

	if v, ok := f[0].(string); ok {
		drate = v
	}
	if v, ok := f[1].(string); ok {
		wrate = v
	}

	res["deposit_usdt_rate"] = drate
	res["withdraw_usdt_rate"] = wrate
	res["name"] = "USDT"
	res["protocol"] = "TRC20"
	return res, nil
}

// USDT 线下USDT支付
func UsdtPay(fctx *fasthttp.RequestCtx, pid, amount, rate, addr, protocolType, hashID string) (string, error) {

	user, err := MemberCache(fctx)
	if err != nil {
		return "", err
	}
	cd, err := ConfigDetail()
	if err != nil {
		return "", pushLog(err, helper.DBErr)
	}
	//存款订单配置开启
	dts := cd["deposit_time_switch"]
	levelLimit := cd["deposit_level_limit"]
	dll, _ := decimal.NewFromString(levelLimit)
	if dts == "1" && decimal.NewFromInt(int64(user.Level)).LessThan(dll) {
		//是否在豁免名单里
		mcl, _ := MemberConfigList("1", user.Username)
		if len(mcl) == 0 {
			dtss := cd["deposit_third_switch"]
			ex1 := g.Ex{"uid": user.UID, "state": g.Op{"neq": DepositSuccess}, "created_at": g.Op{"gte": time.Now().Unix() - 18000}}
			if dtss == "2" {
				ex1["flag"] = []int{3, 4}
			}
			//查最近30分钟有多少条
			total := dataTotal{}
			countQuery, _, _ := dialect.From("tbl_deposit").Select(g.COUNT(1).As("t"), g.MAX("created_at").As("l")).Where(
				ex1).ToSQL()
			err = meta.MerchantDB.Get(&total, countQuery)
			fmt.Println(countQuery)
			if err != nil {
				return "data", pushLog(err, helper.DBErr)
			}
			//有未成功的不能在提交
			dcr := cd["deposit_can_repeat"]
			if dcr == "1" {
				if total.T.Int64 > 1 {
					return "", errors.New(helper.EmptyOrder30MinsBlock)
				}
			}
			depositTimeThreeMax := cd["deposit_time_three_max"]
			depositTimeThree := cd["deposit_time_three"]
			dttma, _ := strconv.ParseInt(depositTimeThreeMax, 10, 64)
			dttb, _ := strconv.Atoi(depositTimeThree)
			if total.T.Int64 >= dttma {
				tts := time.Now().Unix() - total.L.Int64
				if tts < int64(dttb) {
					return "", errors.New(fmt.Sprintf("please wait %d sec", int64(dttb)-tts))
				}
			}
			depositTimeTwoMax := cd["deposit_time_two_max"]
			depositTimeTwoMin := cd["deposit_time_two_min"]
			depositTimeTwo := cd["deposit_time_two"]
			dt2a, _ := strconv.ParseInt(depositTimeTwoMax, 10, 64)
			dt2i, _ := strconv.ParseInt(depositTimeTwoMin, 10, 64)
			dtta, _ := strconv.Atoi(depositTimeTwo)
			if total.T.Int64 >= dt2i && total.T.Int64 <= dt2a {
				tts := time.Now().Unix() - total.L.Int64
				if tts < int64(dtta) {
					return "", errors.New(fmt.Sprintf("please wait %d sec", int64(dtta)-tts))
				}
			}
			dtom := cd["deposit_time_one_max"]
			dto := cd["deposit_time_one"]
			dtomi, _ := strconv.ParseInt(dtom, 10, 64)
			dtoi, _ := strconv.Atoi(dto)
			if total.T.Int64 >= dtomi {
				tts := time.Now().Unix() - total.L.Int64
				if tts < int64(dtoi) {
					return "", errors.New(fmt.Sprintf("please wait %d sec", int64(dtoi)-tts))
				}
			}
		}
	}
	p, err := CachePayment(pid)
	if err != nil {
		return "", errors.New(helper.ChannelNotExist)
	}

	usdt_info_temp, err := UsdtInfo()
	if err != nil {
		return "", err
	}

	usdt_rate, err := decimal.NewFromString(usdt_info_temp["deposit_usdt_rate"])
	if err != nil {
		return "", errors.New(helper.AmountErr)
	}
	r, _ := decimal.NewFromString(rate)
	if usdt_rate.Cmp(r) != 0 {
		return "", errors.New(helper.ExchangeRateRrr)
	}

	dm, err := decimal.NewFromString(amount)
	if err != nil {
		return "", errors.New(helper.AmountErr)
	}

	// 发起的usdt金额
	usdtAmount := (dm.Mul(decimal.NewFromInt(1000)).Div(usdt_rate)).Truncate(2).String()

	// 生成我方存款订单号
	orderID := helper.GenId()

	// 检查用户的存款行为是否过于频繁
	err = cacheDepositProcessing(user.UID, time.Now().Unix())
	if err != nil {
		return "", err
	}
	ca := fctx.Time().In(loc).Unix()
	sn := fmt.Sprintf(`deposit%s%s%d%d`, orderID, user.Username, ca, user.CreatedAt)
	mhash := fmt.Sprintf("%d", cityhash.CityHash64([]byte(sn)))
	d := g.Record{
		"id":                orderID,
		"prefix":            meta.Prefix,
		"oid":               orderID,
		"uid":               user.UID,
		"top_uid":           user.TopUid,
		"top_name":          user.TopName,
		"parent_name":       user.ParentName,
		"parent_uid":        user.ParentUid,
		"username":          user.Username,
		"channel_id":        p.ChannelID,
		"cid":               p.CateID,
		"pid":               p.ID,
		"amount":            amount,
		"state":             DepositConfirming,
		"finance_type":      helper.TransactionUSDTOfflineDeposit,
		"automatic":         "0",
		"created_at":        ca,
		"created_uid":       "0",
		"created_name":      "",
		"confirm_at":        "0",
		"confirm_uid":       "0",
		"confirm_name":      "",
		"review_remark":     "",
		"protocol_type":     protocolType,
		"address":           addr,
		"usdt_apply_amount": usdtAmount,
		"rate":              usdt_rate.String(),
		"hash_id":           hashID,
		"flag":              DepositFlagUSDT,
		"level":             user.Level,
		"r":                 mhash,
		"first_deposit_at":  user.FirstDepositAt,
	}

	// 请求成功插入订单
	err = deposit(d)
	if err != nil {
		fmt.Println("UsdtPay deposit insert into table error: ", err.Error())
		return "", errors.New(helper.DBErr)
	}

	// 记录存款行为
	_ = cacheDepositProcessingInsert(user.UID, orderID, fctx.Time().In(loc).Unix())

	return orderID, nil
}

func UsdtByCol(val string) (VirtualWallet_t, error) {

	var bc VirtualWallet_t
	ex := g.Ex{
		"wallet_addr": val,
	}
	query, _, _ := dialect.From("f_virtual_wallet").Select(colsVirtualWallet...).Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&bc, query)
	if err != nil && err != sql.ErrNoRows {
		return bc, pushLog(err, helper.DBErr)
	}

	if err == sql.ErrNoRows {
		return bc, errors.New(helper.RecordNotExistErr)
	}

	return bc, nil
}

// 添加usdt收款账号
func UsdtInsert(recs VirtualWallet_t, code, adminName string) error {

	if recs.State == "1" {
		d, err := VirtualWalletList(g.Ex{"state": 1}, 1, 10)
		if err != nil {
			return pushLog(err, helper.DBErr)
		}
		if d.T > 0 {
			return errors.New(helper.CanOnlyOpenOnePayeeAccount)
		}
	}

	query, _, _ := dialect.Insert("f_virtual_wallet").Rows(recs).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}
	VirtualWalletUpdateCache()

	contentLog := fmt.Sprintf("渠道管理-线下usdt收款地址-新增:后台账号:%s【id:%s,usdt地址:%s,最大充值金额:%f,最小充值:%f】",
		adminName, recs.Id, recs.WalletAddr, recs.MaxAmount, recs.MinAmount)
	AdminLogInsert(ChannelModel, contentLog, InsertOp, adminName)

	return nil
}

// UsdtDelete
func UsdtDelete(id, adminName string) error {

	vw, err := UsdtByID(id)
	if err != nil {
		return err
	}

	ex := g.Ex{
		"id": id,
	}
	query, _, _ := dialect.Delete("f_virtual_wallet").Where(ex).ToSQL()
	_, err = meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}
	VirtualWalletUpdateCache()

	contentLog := fmt.Sprintf("渠道管理-线下USDT-新增:后台账号:%s【id:%s,USDT地址:%s】",
		adminName, id, vw.WalletAddr)
	AdminLogInsert(ChannelModel, contentLog, DeleteOp, adminName)

	return nil
}

func VirtualWalletUpdate(id string, record g.Record) error {

	ex := g.Ex{
		"id": id,
	}
	query, _, _ := dialect.Update("f_virtual_wallet").Set(record).Where(ex).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}
	VirtualWalletUpdateCache()
	return nil
}

func UsdtByID(id string) (VirtualWallet_t, error) {

	var bc VirtualWallet_t
	ex := g.Ex{
		"id": id,
	}
	query, _, _ := dialect.From("f_virtual_wallet").Select(colsVirtualWallet...).Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&bc, query)
	if err != nil && err != sql.ErrNoRows {
		return bc, pushLog(err, helper.DBErr)
	}

	if err == sql.ErrNoRows {
		return bc, errors.New(helper.BankCardNotExist)
	}

	return bc, nil
}

func VirtualWalletUpdateCache() error {

	usdt_info_temp, err := UsdtInfo()
	if err != nil {
		return err
	}
	usdt_rate, err := decimal.NewFromString(usdt_info_temp["deposit_usdt_rate"])
	if err != nil {
		return err
	}
	key := meta.Prefix + ":offline:usdt:one"
	ex := g.Ex{
		"state":    1,
		"currency": 1,
	}
	res, err := VirtualWalletList(ex, 1, 10)
	if err != nil {
		fmt.Println("VirtualWalletUpdateCache err = ", err)
		return err
	}

	if len(res.D) == 0 {
		fmt.Println("VirtualWalletUpdateCache len(res) = 0")
		meta.MerchantRedis.Unlink(ctx, key).Err()
		return nil
	}

	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	pipe.Unlink(ctx, key)
	for _, v := range res.D {
		pipe.Set(ctx, key, v.Id, 5*time.Hour)
		value := map[string]interface{}{
			"usdt_rate":   usdt_rate.StringFixed(2),
			"qr_img":      v.QrImg,
			"protocol":    "TRC20",
			"min_amount":  v.MinAmount,
			"max_amount":  v.MaxAmount,
			"wallet_addr": v.WalletAddr,
		}
		vkey := key + ":" + v.Id
		pipe.HMSet(ctx, vkey, value)
		pipe.Persist(ctx, key)
		pipe.Persist(ctx, vkey)

	}
	pipe.Persist(ctx, key)

	_, err = pipe.Exec(ctx)
	if err != nil {
		fmt.Println("BankCardUpdateCache pipe.Exec = ", err)
		return errors.New(helper.RedisErr)
	}

	return nil
}

// 线下虚拟钱包列表
func VirtualWalletList(ex g.Ex, page, pageSize uint) (VirtualWalletData, error) {

	var data VirtualWalletData
	offset := (page - 1) * pageSize

	if page == 1 {
		query, _, _ := dialect.From("f_virtual_wallet").Select(g.COUNT(1)).Where(ex).ToSQL()
		err := meta.MerchantDB.Get(&data.T, query)
		if err != nil {
			return data, pushLog(err, helper.DBErr)
		}

		if data.T == 0 {
			return data, nil
		}
	}

	query, _, _ := dialect.From("f_virtual_wallet").Select(colsVirtualWallet...).Where(ex).Offset(offset).Limit(uint(pageSize)).Order(g.C("sort").Asc(), g.C("created_at").Desc()).ToSQL()
	err := meta.MerchantDB.Select(&data.D, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	return data, nil
}

// UsdtWithdrawUserInsert 用户usdt申请订单
func UsdtWithdrawUserInsert(amount, rate string, fCtx *fasthttp.RequestCtx) (string, error) {

	mb, err := MemberCache(fCtx)
	if err != nil {
		return "", errors.New(helper.AccessTokenExpires)
	}

	usdt_info_temp, err := UsdtInfo()
	if err != nil {
		return "", errors.New(helper.RedisErr)
	}
	if usdt_info_temp["withdraw_usdt_rate"] != rate {
		return "", errors.New(helper.AccessTokenExpires)
	}

	r, _ := decimal.NewFromString(rate)
	wr, _ := decimal.NewFromString(usdt_info_temp["withdraw_usdt_rate"])
	if r.Cmp(wr) != 0 {
		return "", errors.New(helper.ExchangeRateRrr)
	}

	var mVWallet MemberVirtualWallet
	query, _, _ := dialect.From("tbl_member_virtual_wallet").Select(colsMemberVirtualWallet...).Where(g.Ex{"uid": mb.UID}).ToSQL()
	err = meta.MerchantDB.Get(&mVWallet, query)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	// 记录不存在
	if err == sql.ErrNoRows {
		return "", errors.New(helper.VirtualAddrNotExist)
	}

	blkey := fmt.Sprintf("%s:merchant:virtual_blacklist", meta.Prefix)
	ex2Temp := meta.MerchantRedis.SIsMember(ctx, blkey, mVWallet.WalletAddr)
	if ex2Temp.Val() {
		return "", errors.New(helper.WithdrawBan)
	}

	var vw VirtualWallet_t

	ex := g.Ex{
		"state": "1",
	}
	query, _, _ = dialect.From("f_virtual_wallet").Select(colsVirtualWallet...).Where(ex).ToSQL()
	err = meta.MerchantDB.Get(&vw, query)
	if err != nil {
		return "", errors.New(helper.NoPayChannel)
	}

	withdrawAmount, err := decimal.NewFromString(amount)
	if err != nil {
		return "", errors.New(helper.FormatErr)
	}

	fmin := decimal.NewFromFloat(vw.MinAmount)
	if fmin.Cmp(withdrawAmount) > 0 {
		return "", errors.New(helper.AmountErr)
	}

	fmax := decimal.NewFromFloat(vw.MaxAmount)
	if fmax.Cmp(withdrawAmount) < 0 {
		return "", errors.New(helper.AmountErr)
	}

	// 检查上次提现成功到现在的存款流水是否满足 未满足的返回流水未达标
	_, err = ryrpc.CheckDepositFlow(mb.Username)
	if err != nil {
		fmt.Println("查询某个用户的流水:", err)
		return "", err
	}

	//查询今日提款总计
	count, totalAmount, err := withdrawDailyData(mb.Username)
	if err != nil {
		return "", errors.New(helper.ServerErr)
	}

	// 所属vip提现次数限制
	timesKey := fmt.Sprintf("%s:vip:withdraw:maxtimes", meta.Prefix)
	times, err := meta.MerchantRedis.HGet(ctx, timesKey, fmt.Sprintf(`%d`, mb.Level)).Result()
	if err != nil {
		return "", pushLog(err, helper.RedisErr)
	}

	num, err := strconv.ParseInt(times, 10, 64)
	if err != nil {
		return "", pushLog(err, helper.FormatErr)
	}

	//今日提款次数大于等于所属vip提现次数限制
	if count >= num {
		return "", errors.New(helper.DailyTimesLimitErr)
	}

	// 所属vip提现金额限制
	amountKey := fmt.Sprintf("%s:vip:withdraw:maxamount", meta.Prefix)
	maxAmount, err := meta.MerchantRedis.HGet(ctx, amountKey, fmt.Sprintf(`%d`, mb.Level)).Result()
	if err != nil {
		return "", pushLog(err, helper.RedisErr)
	}

	max, err := decimal.NewFromString(maxAmount)
	if err != nil {
		return "", pushLog(err, helper.FormatErr)
	}

	//当前提现金额 大于 所属等级每日提现金额限制
	if withdrawAmount.Cmp(max) > 0 {
		return "", errors.New(helper.MaxDrawLimitParamErr)
	}

	// 今日已经申请的提现金额大于所属等级每日提现金额限制 或者 今日已经申请的提现金额加上当前提现金额大于所属等级每日提现金额限制
	if totalAmount.Cmp(max) > 0 || totalAmount.Add(withdrawAmount).Cmp(max) > 0 {
		return "", errors.New(helper.DailyAmountLimitErr)
	}

	var (
		receiveAt  int64
		withdrawId = helper.GenLongId()
		state      = WithdrawReviewing
		adminName  string
	)

	if mb.Tester == "0" {
		state = WithdrawSuccess
	}
	// 记录提款单
	err = usdtWithdrawInsert(mVWallet, amount, rate, withdrawId, "0", adminName, receiveAt, state, fCtx.Time(), mb)
	if err != nil {
		return "", err
	}

	if mb.Tester == "1" {

		// 发送消息通知
		_ = pushWithdrawNotify(withdrawReviewFmt, mb.Username, amount)
	}

	if mb.Tester == "0" {
		//发送推送
		msg := fmt.Sprintf(`{"ty":"2","amount": "%s", "ts":"%d","status":"success"}`, amount, time.Now().Unix())
		//fmt.Println(msg)
		topic := fmt.Sprintf("%s/%s/finance", meta.Prefix, mb.UID)
		err = Publish(topic, []byte(msg))
		if err != nil {
			fmt.Println("merchantNats.Publish finance = ", err.Error())
		}
	}

	return withdrawId, nil
}

func usdtWithdrawInsert(mv MemberVirtualWallet, amount, rate, withdrawID, confirmUid, confirmName string, receiveAt int64, state int, ts time.Time, member Member) error {

	// lock and defer unlock
	lk := fmt.Sprintf("w:%s", member.Username)
	err := withLock(lk)
	if err != nil {
		return err
	}

	//defer Unlock(lk)

	// 同时只能有一笔提款在处理中
	ex := g.Ex{
		"uid":   member.UID,
		"state": g.Op{"notIn": []int64{WithdrawReviewReject, WithdrawSuccess, WithdrawFailed}},
	}

	err = withdrawOrderExists(ex)
	if err != nil {
		return err
	}

	withdrawAmount, err := decimal.NewFromString(amount)
	if err != nil {
		return pushLog(err, helper.AmountErr)
	}

	// check balance
	userAmount, err := BalanceIsEnough(member.UID, withdrawAmount)
	if err != nil {
		return err
	}

	lastDeposit, err := depositLastAmount(member.UID)
	if err != nil {
		return err
	}

	withdrawRate, err := decimal.NewFromString(rate)
	if err != nil {
		return err
	}
	fmt.Println(withdrawAmount)
	fmt.Println(withdrawRate)
	virtualCount := withdrawAmount.Mul(decimal.NewFromInt(1000)).Div(withdrawRate).Truncate(2).StringFixed(2)
	fmt.Println(virtualCount)

	// 默认取代代付
	automatic := 0
	sn := fmt.Sprintf(`withdraw%s%s%d%d`, withdrawID, member.Username, ts.Unix(), member.CreatedAt)
	mhash := fmt.Sprintf("%d", cityhash.CityHash64([]byte(sn)))
	record := g.Record{
		"id":                  withdrawID,
		"prefix":              meta.Prefix,
		"bid":                 mv.ID,
		"flag":                2,
		"oid":                 withdrawID,
		"uid":                 member.UID,
		"top_uid":             member.TopUid,
		"top_name":            member.TopName,
		"parent_name":         member.ParentName,
		"parent_uid":          member.ParentUid,
		"username":            member.Username,
		"pid":                 779402438062874465,
		"amount":              withdrawAmount.Truncate(4).String(),
		"state":               state,
		"automatic":           automatic, //1:自动转账  0:人工确认
		"created_at":          ts.Unix(),
		"finance_type":        helper.TransactionWithDraw,
		"real_name_hash":      member.RealnameHash,
		"last_deposit_amount": lastDeposit,
		"receive_at":          receiveAt,
		"confirm_uid":         confirmUid,
		"confirm_name":        confirmName,
		"wallet_flag":         1,
		"level":               member.Level,
		"tester":              member.Tester,
		"balance":             userAmount.Sub(withdrawAmount).String(),
		"r":                   mhash,
		"virtual_count":       virtualCount,
		"virtual_rate":        rate,
		"currency":            1,
		"protocol":            1,
		"alias":               mv.Alias,
		"wallet_addr":         mv.WalletAddr,
		"last_withdraw_at":    member.LastWithdrawAt,
	}

	// 开启事务 写账变 更新redis  查询提款
	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	query, _, _ := dialect.Insert("tbl_withdraw").Rows(record).ToSQL()
	fmt.Println(query)
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	if member.Tester == "1" {
		// 更新余额
		ex = g.Ex{
			"uid":    member.UID,
			"prefix": meta.Prefix,
		}
		balanceRecord := g.Record{
			"balance":     g.L(fmt.Sprintf("balance-%s", withdrawAmount.String())),
			"lock_amount": g.L(fmt.Sprintf("lock_amount+%s", withdrawAmount.String())),
		}
		query, _, _ = dialect.Update("tbl_members").Set(balanceRecord).Where(ex).ToSQL()
		_, err = tx.Exec(query)
		if err != nil {
			_ = tx.Rollback()
			return pushLog(err, helper.DBErr)
		}

		// 写入账变
		mbTrans := memberTransaction{
			AfterAmount:  userAmount.Sub(withdrawAmount).String(),
			Amount:       withdrawAmount.String(),
			BeforeAmount: userAmount.String(),
			BillNo:       withdrawID,
			CreatedAt:    ts.UnixNano() / 1e6,
			ID:           helper.GenId(),
			CashType:     helper.TransactionWithDraw,
			UID:          member.UID,
			Username:     member.Username,
			Prefix:       meta.Prefix,
		}

		query, _, _ = dialect.Insert("tbl_balance_transaction").Rows(mbTrans).ToSQL()
		_, err = tx.Exec(query)
		if err != nil {
			_ = tx.Rollback()
			return pushLog(err, helper.DBErr)
		}
	} else {
		// 更新余额
		ex = g.Ex{
			"uid":    member.UID,
			"prefix": meta.Prefix,
		}
		balanceRecord := g.Record{
			"balance": g.L(fmt.Sprintf("balance-%s", withdrawAmount.String())),
		}
		query, _, _ = dialect.Update("tbl_members").Set(balanceRecord).Where(ex).ToSQL()
		_, err = tx.Exec(query)
		if err != nil {
			_ = tx.Rollback()
			return pushLog(err, helper.DBErr)
		}

		// 写入账变
		mbTrans := memberTransaction{
			AfterAmount:  userAmount.Sub(withdrawAmount).String(),
			Amount:       withdrawAmount.String(),
			BeforeAmount: userAmount.String(),
			BillNo:       withdrawID,
			CreatedAt:    ts.UnixNano() / 1e6,
			ID:           helper.GenId(),
			CashType:     helper.TransactionWithDraw,
			UID:          member.UID,
			Username:     member.Username,
			Prefix:       meta.Prefix,
		}

		query, _, _ = dialect.Insert("tbl_balance_transaction").Rows(mbTrans).ToSQL()
		_, err = tx.Exec(query)
		if err != nil {
			_ = tx.Rollback()
			return pushLog(err, helper.DBErr)
		}
	}

	err = tx.Commit()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}
	MemberUpdateCache(member.Username)
	return nil
}
