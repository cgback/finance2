package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"finance/contrib/validator"
	ryrpc "finance/rpc"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/go-redis/redis/v8"
	"github.com/shopspring/decimal"
	"github.com/spaolacci/murmur3"
	"github.com/tenfyzhong/cityhash"
	"github.com/valyala/fasthttp"
	"strconv"
	"strings"
	"time"
)

type withdrawTotal struct {
	T   sql.NullInt64   `json:"t"`
	Agg sql.NullFloat64 `json:"agg"`
}

// 今日提款成功次数和金额
func withdrawDailyData(username string) (int64, decimal.Decimal, error) {

	data := withdrawTotal{}
	ex := g.Ex{
		"prefix":     meta.Prefix,
		"username":   username,
		"state":      WithdrawSuccess,
		"created_at": g.Op{"between": exp.NewRangeVal(helper.DayTST(0, loc).Unix(), helper.DayTET(0, loc).Unix())},
	}
	query, _, _ := dialect.From("tbl_withdraw").Select(g.COUNT("id").As("t"), g.SUM("amount").As("agg")).Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&data, query)

	if err != nil && err != sql.ErrNoRows {
		return 0, decimal.Zero, pushLog(err, helper.DBErr)
	}

	return data.T.Int64, decimal.NewFromFloat(data.Agg.Float64), nil
}

// 检查订单是否存在
func withdrawOrderExists(ex g.Ex) error {

	var id string
	query, _, _ := dialect.From("tbl_withdraw").Select("id").Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&id, query)

	if err != nil && err != sql.ErrNoRows {
		return pushLog(err, helper.DBErr)
	}

	if id != "" {
		return errors.New(helper.OrderProcess)
	}

	return nil
}

// WithdrawUserInsert 用户申请订单
func WithdrawUserInsert(amount, bid, sid, ts, verifyCode string, fCtx *fasthttp.RequestCtx) (string, error) {

	mb, err := MemberCache(fCtx)
	if err != nil {
		return "", errors.New(helper.AccessTokenExpires)
	}

	recs, err := ryrpc.KmsDecryptOne(mb.UID, false, []string{"phone"})
	if err != nil {
		fmt.Println(err)
		_ = pushLog(err, helper.GetRPCErr)
		return "", errors.New(helper.GetRPCErr)
	}

	//如果开启了提现验证码就要检测。每天首次提现要校验验证码
	if mb.Tester == "1" {
		rs, err := checkWithdrawSms(mb, recs["phone"], sid, ts, verifyCode, fCtx)
		if err != nil {
			return rs, errors.New(helper.FirstDailyWithdrawNeedVerify)
		}
	}

	var bankcardHash uint64
	query, _, _ := dialect.From("tbl_member_bankcard").Select("bank_card_hash").Where(g.Ex{"id": bid, "state": 1}).ToSQL()
	err = meta.MerchantDB.Get(&bankcardHash, query)
	if err != nil {
		return "", err
	}

	// 记录不存在
	if bankcardHash == 0 {
		return "", errors.New(helper.RecordNotExistErr)
	}
	pt, err := ChanByID("133221087319615487")
	if err != nil {
		return "", errors.New(helper.NoPayChannel)
	}

	if len(pt.VipList) == 0 {
		return "", errors.New(helper.NoPayChannel)
	}

	withdrawAmount, err := decimal.NewFromString(amount)
	if err != nil {
		return "", errors.New(helper.FormatErr)
	}

	fmin, _ := decimal.NewFromString(pt.Fmin)
	if fmin.Cmp(withdrawAmount) > 0 {
		return "", errors.New(helper.AmountErr)
	}

	fmax, _ := decimal.NewFromString(pt.Fmax)
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

	// 获取风控UID
	uid, err := GetRisksUID()
	if err != nil {
		//fmt.Println("风控人员未找到: 订单id=", withdrawId, "err:", err)
		uid = "0"
	}

	if uid != "0" {
		// 获取风控审核人的name
		adminName, err = AdminGetName(uid)
		if err != nil {
			return "", err
		}

		if adminName == "" {
			fmt.Println("风控人员未找到: 订单id=", withdrawId, "uid:", uid, "err:", err)
			uid = "0"
		}

		if uid != "0" {
			state = WithdrawDispatched
			receiveAt = fCtx.Time().Unix()
		}
	}
	if mb.Tester == "0" {
		state = WithdrawSuccess
	}
	// 记录提款单
	err = WithdrawInsert(amount, bid, withdrawId, uid, adminName, receiveAt, state, fCtx.Time(), mb)
	if err != nil {
		return "", err
	}

	if uid != "0" {
		_ = SetRisksOrder(uid, withdrawId, 1)
	} else {
		/*
			// 自动派单模式
			exist, _ := meta.MerchantRedis.Get(ctx, risksState).Result()
			if exist == "1" {
				// 无风控人员可以分配
				param := map[string]interface{}{
					"id": withdrawId,
				}
				_, _ = BeanPut("risk", param, 10)
			}
		*/
	}
	if mb.Tester == "1" {
		// 发送消息通知
		_ = PushWithdrawNotify(withdrawReviewFmt, mb.Username, amount)
	} else if mb.Tester == "0" {
		//发送推送
		msg := fmt.Sprintf(`{"ty":"2","amount": "%s", "ts":"%d","status":"success"}`, amount, time.Now().Unix())
		topic := fmt.Sprintf("%s/%s/finance", meta.Prefix, mb.UID)
		err = Publish(topic, []byte(msg))
		if err != nil {
			fmt.Println("merchantNats.Publish finance = ", err.Error())
		}
	}

	return withdrawId, nil
}

func checkWithdrawSms(mb Member, phone, sid, ts, verifyCode string, fCtx *fasthttp.RequestCtx) (string, error) {

	// 有就是每日提款要验证码
	key := fmt.Sprintf("%s:sms:enablemod", meta.Prefix)
	cmd := meta.MerchantRedis.Exists(ctx, key)
	fmt.Println("WithdrawUserInsert", mb.Username, cmd.String())
	if cmd.Val() == 1 {
		// 每日提款redis记录
		key = fmt.Sprintf("%s:fianance:withdraw:daily:%s", meta.Prefix, mb.Username)
		dailyCmd := meta.MerchantRedis.Exists(ctx, key)
		fmt.Println("WithdrawUserInsert", mb.Username, dailyCmd.String())
		// 每日第一次提款
		if mb.Tester == "1" && 0 == dailyCmd.Val() {

			if !validator.CtypeDigit(sid) || //短信验证码id
				!validator.CtypeDigit(ts) || //短信记录ts
				verifyCode == "" { //验证码校验
				return "", errors.New(helper.FirstDailyWithdrawNeedVerify)
			}

			ip := helper.FromRequest(fCtx)
			ok, err := CheckSmsCaptcha(ip, ts, sid, phone, verifyCode)
			if err != nil || !ok {
				return "", errors.New(helper.PhoneVerificationErr)
			}

			y, m, d := fCtx.Time().Date()
			pipe := meta.MerchantRedis.TxPipeline()
			defer pipe.Close()

			pipe.Set(ctx, key, 1, 1*time.Hour)
			pipe.ExpireAt(ctx, key, time.Date(y, m, d, 23, 59, 59, 0, loc))
			_, err = pipe.Exec(ctx)
			if err != nil {
				return "", pushLog(err, helper.RedisErr)
			}
		}
	} else if mb.LastWithdrawAt == 0 {

		//没有开启的就是查有提款成功或者redis有就不用提示
		key = fmt.Sprintf("%s:fianance:withdraw:week:%s", meta.Prefix, mb.Username)
		cmd = meta.MerchantRedis.Exists(ctx, key)
		if mb.LastWithdrawAt != 0 || cmd.Val() == 1 {

		} else {
			if !validator.CtypeDigit(sid) || //短信验证码id
				!validator.CtypeDigit(ts) || //短信记录ts
				verifyCode == "" { //验证码校验
				return "", errors.New(helper.FirstDailyWithdrawNeedVerify)
			}

			ip := helper.FromRequest(fCtx)
			ok, err := CheckSmsCaptcha(ip, ts, sid, phone, verifyCode)
			if err != nil || !ok {
				return "", errors.New(helper.PhoneVerificationErr)
			}

			fmt.Println("WithdrawUserInsert LastWithdrawAt == 0", mb.Username, meta.MerchantRedis.Set(ctx, key, 1, 7*24*time.Hour).String())
		}

	}
	return "", nil
}

func WithdrawInsert(amount, bid, withdrawID, confirmUid, confirmName string, receiveAt int64, state int, ts time.Time, member Member) error {

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

	// 判断银行卡
	ex = g.Ex{
		"uid":   member.UID,
		"id":    bid,
		"state": 1,
	}
	exist := BankCardExist(ex)
	if !exist {
		return errors.New(helper.BankCardNotExist)
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

	cd, err := ConfigDetail()
	if err != nil {
		return err
	}
	withdraw_auto_min := cd["withdraw_auto_min"]
	// 默认取代代付
	//automatic := 1
	//// 根据金额判断 该笔提款是否走代付渠道
	automatic := 0
	wam, _ := decimal.NewFromString(withdraw_auto_min)
	if withdrawAmount.LessThanOrEqual(wam) && member.LastWithdrawAt != 0 {
		state = WithdrawDealing
	}
	mcl, _ := MemberConfigList("1", member.Username)
	if len(mcl) > 0 {
		state = WithdrawReviewing
	}
	sn := fmt.Sprintf(`withdraw%s%s%d%d`, withdrawID, member.Username, ts.Unix(), member.CreatedAt)
	mhash := fmt.Sprintf("%d", cityhash.CityHash64([]byte(sn)))
	record := g.Record{
		"id":                  withdrawID,
		"prefix":              meta.Prefix,
		"bid":                 bid,
		"flag":                1,
		"oid":                 withdrawID,
		"uid":                 member.UID,
		"top_uid":             member.TopUid,
		"top_name":            member.TopName,
		"parent_name":         member.ParentName,
		"parent_uid":          member.ParentUid,
		"username":            member.Username,
		"pid":                 0,
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
		"wallet_flag":         "1",
		"level":               member.Level,
		"tester":              member.Tester,
		"balance":             userAmount.Sub(withdrawAmount).String(),
		"r":                   mhash,
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

func BankCardExist(ex g.Ex) bool {

	var id string
	t := dialect.From("tbl_member_bankcard")
	query, _, _ := t.Select("uid").Where(ex).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&id, query)
	if err == sql.ErrNoRows {
		return false
	}

	return true
}

// 提款历史记录
func WithdrawHistoryList(ex g.Ex, ty, startTime, endTime, isAsc, sortField string, page, pageSize uint) (FWithdrawData, error) {

	ex["prefix"] = meta.Prefix
	ex["tester"] = "1"
	orderField := g.L("created_at")
	if sortField != "" {
		orderField = g.L(sortField)
	}

	orderBy := orderField.Desc()
	if isAsc == "1" {
		orderBy = orderField.Asc()
	}
	if startTime != "" && endTime != "" {

		startAt, err := helper.TimeToLoc(startTime, loc)
		if err != nil {
			return FWithdrawData{}, errors.New(helper.DateTimeErr)
		}

		endAt, err := helper.TimeToLoc(endTime, loc)
		if err != nil {
			return FWithdrawData{}, errors.New(helper.DateTimeErr)
		}

		if startAt >= endAt {
			return FWithdrawData{}, errors.New(helper.QueryTimeRangeErr)
		}

		if ty == "1" {
			ex["created_at"] = g.Op{"between": exp.NewRangeVal(startAt, endAt)}
		} else {
			ex["withdraw_at"] = g.Op{"between": exp.NewRangeVal(startAt, endAt)}
		}
	}

	if realName, ok := ex["real_name_hash"]; ok {
		ex["real_name_hash"] = MurmurHash(realName.(string), 0)
	}

	data := FWithdrawData{}
	if page == 1 {
		query, _, _ := dialect.From("tbl_withdraw").Select(g.COUNT("id")).Where(ex).ToSQL()
		err := meta.MerchantDB.Get(&data.T, query)
		if err != nil {
			return data, pushLog(err, helper.DBErr)
		}

		if data.T == 0 {
			return data, nil
		}

		query, _, _ = dialect.From("tbl_withdraw").Select(g.SUM("amount").As("amount")).Where(ex).ToSQL()
		err = meta.MerchantDB.Get(&data.Agg, query)
		if err != nil {
			return data, pushLog(err, helper.DBErr)
		}
	}
	offset := (page - 1) * pageSize
	query, _, _ := dialect.From("tbl_withdraw").Select(colsWithdraw...).Where(ex).
		Offset(offset).Limit(pageSize).Order(orderBy, g.C("id").Desc()).ToSQL()
	err := meta.MerchantDB.Select(&data.D, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	return data, nil
}

func MurmurHash(str string, seed uint32) uint64 {

	h64 := murmur3.New64WithSeed(seed)
	h64.Write([]byte(str))
	v := h64.Sum64()
	h64.Reset()

	return v
}

// 处理 提款订单返回数据
func WithdrawDealListData(data FWithdrawData) (WithdrawListData, error) {

	result := WithdrawListData{
		T:   data.T,
		Agg: data.Agg,
	}

	if len(data.D) == 0 {
		return result, nil
	}

	// 获取渠道号的pid slice
	pids := make([]string, 0)
	var agencyNames []string
	// 组装获取rpc数据参数
	rpcParam := make(map[string][]string)
	namesMap := make(map[string]string)
	for _, v := range data.D {
		rpcParam["bankcard"] = append(rpcParam["bankcard"], v.BID)
		rpcParam["realname"] = append(rpcParam["realname"], v.UID)
		namesMap[v.Username] = v.UID
		pids = append(pids, v.PID)

		if v.ParentName != "" && v.ParentName != "root" {
			agencyNames = append(agencyNames, v.ParentName)
		}

	}
	userMap := map[string]MBBalance{}
	var uids []string
	if len(data.D) > 0 {

		for _, v := range data.D {
			uids = append(uids, v.UID)
		}

		balances, err := getBalanceByUids(uids)
		if err != nil {
			return result, err
		}

		for _, v := range balances {
			userMap[v.UID] = v
		}
	}

	// 遍历用户map 读取标签数据
	var names []string
	tags := make(map[string]string)
	for name, uid := range namesMap {
		// 获取用户标签
		memberTag, err := MemberTagsList(uid)
		if err != nil {
			return result, err
		}
		// 组装需要通过name获取的 redis参数
		names = append(names, name)
		tags[name] = memberTag
	}

	bankcards, err := bankcardListDBByIDs(rpcParam["bankcard"])
	if err != nil {
		return result, err
	}

	encFields := []string{"realname"}

	for _, v := range rpcParam["bankcard"] {
		encFields = append(encFields, "bankcard"+v)
	}

	recs, err := ryrpc.KmsDecryptAll(rpcParam["realname"], false, encFields)
	if err != nil {
		_ = pushLog(err, helper.GetRPCErr)
		return result, errors.New(helper.GetRPCErr)
	}

	cids, _ := channelCateMap(pids)
	wm, err := withdrawFirst(uids)
	if err != nil {
		return result, err
	}
	// 处理返回前端的数据
	for _, v := range data.D {

		wat := wm[v.UID]
		w := withdrawCols{
			mWithdraw:          v,
			MemberBankNo:       recs[v.UID]["bankcard"+v.BID],
			MemberBankRealName: recs[v.UID]["realname"],
			MemberRealName:     recs[v.UID]["realname"],
			MemberTags:         tags[v.Username],
			Balance:            v.Balance,
			LockAmount:         userMap[v.UID].LockAmount,
		}
		if wat > 0 && wat != v.CreatedAt {
			w.FirstWithdraw = false
		} else if wat == v.CreatedAt || wat == 0 {
			w.FirstWithdraw = true
		}

		// 匹配银行卡信息
		card, ok := bankcards[v.BID]
		if ok {
			w.MemberBankName = card.BankID
			w.MemberBankAddress = card.BankAddress
		}

		// 匹配渠道信息
		cate, ok := cids[v.PID]
		if ok {
			w.CateID = cate.ID
			w.CateName = cate.Name
		}

		result.D = append(result.D, w)
	}

	return result, nil
}

// WithdrawList 提款记录
func WithdrawList(ex g.Ex, ty uint8, startTime, endTime string, isBig, firstWd int, page, pageSize uint) (FWithdrawData, error) {

	ex["prefix"] = meta.Prefix

	if startTime != "" && endTime != "" {

		startAt, err := helper.TimeToLoc(startTime, loc)
		if err != nil {
			return FWithdrawData{}, errors.New(helper.DateTimeErr)
		}

		endAt, err := helper.TimeToLoc(endTime, loc)
		if err != nil {
			return FWithdrawData{}, errors.New(helper.DateTimeErr)
		}

		if ty == 1 {
			ex["created_at"] = g.Op{"between": exp.NewRangeVal(startAt, endAt)}
		} else {
			ex["withdraw_at"] = g.Op{"between": exp.NewRangeVal(startAt, endAt)}
		}
	}
	// 待派单特殊操作 只显示一天的数据
	if ty == 3 {
		now := time.Now().Unix()
		ex["created_at"] = g.Op{"between": exp.NewRangeVal(now-172800, now)}
	}

	if realName, ok := ex["real_name_hash"]; ok {
		ex["real_name_hash"] = fmt.Sprintf("%d", MurmurHash(realName.(string), 0))
	}

	var data FWithdrawData
	if page == 1 {
		var total withdrawTotal
		query, _, _ := dialect.From("tbl_withdraw").Select(g.COUNT(1).As("t"), g.SUM("amount").As("agg")).Where(ex).ToSQL()
		//fmt.Println(query)
		err := meta.MerchantDB.Get(&total, query)
		if err != nil {
			return data, pushLog(err, helper.DBErr)
		}

		data.T = total.T.Int64
		data.Agg = Withdraw{
			Amount: total.Agg.Float64,
		}
	}
	orderTemp := "created_at"
	cols := colsWithdraw
	cd, err := ConfigDetail()
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}
	depositListFirst := cd["withdraw_list_first"]
	if isBig == 1 && firstWd == 1 {
		cols = append(cols, g.L("case when last_withdraw_at = 0 then 2000000000+created_at  when amount > "+depositListFirst+" then (created_at+1000000000) else created_at end as sort_num"))
		orderTemp = "sort_num"
	}
	if isBig == 1 {
		cols = append(cols, g.L("case when amount > "+depositListFirst+" then (created_at+1000000000) else created_at end as sort_num"))
		orderTemp = "sort_num"
	}
	if firstWd == 1 {
		cols = append(cols, g.L("case when last_withdraw_at = 0 then 2000000000+created_at else created_at end as sort_num"))
		orderTemp = "sort_num"
	}
	offset := (page - 1) * pageSize
	query, _, _ := dialect.From("tbl_withdraw").
		Select(cols...).Where(ex).Order(g.C(orderTemp).Desc()).Offset(offset).Limit(pageSize).ToSQL()
	//fmt.Println(query)
	err = meta.MerchantDB.Select(&data.D, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	return data, nil
}

func MemberTagsList(uid string) (string, error) {

	var tags []string
	ex := g.Ex{"uid": uid}
	query, _, _ := dialect.From("tbl_member_tags").Select("tag_name").Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&tags, query)
	if err != nil {
		return "", pushLog(err, helper.DBErr)
	}

	return strings.Join(tags, ","), nil
}

func bankcardListDBByIDs(ids []string) (map[string]MemberBankCard, error) {

	data := make(map[string]MemberBankCard)
	if len(ids) == 0 {
		return nil, errors.New(helper.UsernameErr)
	}

	ex := g.Ex{"id": ids}
	bankcards, err := MemberBankcardList(ex)
	if err != nil {
		return data, err
	}

	for _, v := range bankcards {
		data[v.ID] = v
	}

	return data, nil
}

func MemberBankcardList(ex g.Ex) ([]MemberBankCard, error) {

	var data []MemberBankCard
	ex["prefix"] = meta.Prefix
	t := dialect.From("tbl_member_bankcard")
	query, _, _ := t.Select(colsMemberBankcard...).Where(ex).Order(g.C("created_at").Desc()).ToSQL()
	err := meta.MerchantDB.Select(&data, query)
	if err != nil && err != sql.ErrNoRows {
		return data, pushLog(err, helper.DBErr)
	}

	return data, nil
}

func withdrawFirst(uids []string) (map[string]int64, error) {

	wm := map[string]int64{}
	var w []Withdraw
	query, _, _ := dialect.From("tbl_withdraw").Select(g.C("uid"), g.MIN("created_at").As("created_at")).Where(g.Ex{"uid": uids, "state": WithdrawSuccess}).GroupBy(g.C("uid")).ToSQL()
	err := meta.MerchantDB.Select(&w, query)
	if err == sql.ErrNoRows {
		return wm, errors.New(helper.OrderNotExist)
	}

	for _, v := range w {
		wm[v.UID] = v.CreatedAt
	}
	return wm, err
}

func WithdrawUpdateInfo(id string, record g.Record) error {

	query, _, _ := dialect.Update("tbl_withdraw").Where(g.Ex{"id": id}).Set(record).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return nil
}

// 财务拒绝提款订单
func WithdrawReject(id string, d g.Record) error {
	return WithdrawDownPoint(id, "", WithdrawFailed, d)
}

// 取款下分
func WithdrawDownPoint(did, bankcard string, state int, record g.Record) error {

	//判断状态是否合法
	allow := map[int]bool{
		WithdrawReviewReject:  true,
		WithdrawDealing:       true,
		WithdrawSuccess:       true,
		WithdrawFailed:        true,
		WithdrawAutoPayFailed: true,
	}
	if _, ok := allow[state]; !ok {
		return errors.New(helper.StateParamErr)
	}

	//1、判断订单是否存在
	var order Withdraw
	ex := g.Ex{"id": did}
	query, _, _ := dialect.From("tbl_withdraw").Select(colsWithdraw...).Where(ex).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&order, query)
	if err != nil || len(order.Username) < 1 {
		return errors.New(helper.IDErr)
	}

	query, _, _ = dialect.Update("tbl_withdraw").Set(record).Where(ex).ToSQL()
	switch order.State {
	case WithdrawReviewing:
		// 审核中(风控待领取)的订单只能流向分配, 上层业务处理
		return errors.New(helper.OrderStateErr)
	case WithdrawDealing:
		// 出款处理中可以是自动代付失败(WithdrawAutoPayFailed) 提款成功(WithdrawSuccess) 提款失败(WithdrawFailed)
		// 自动代付失败和提款成功是调用三方代付才会有的状态
		// 提款失败通过提款管理的[拒绝]操作进行流转(同时出款类型必须是手动出款)
		if state == WithdrawAutoPayFailed {
			_, err = meta.MerchantDB.Exec(query)
			if err != nil {
				return pushLog(err, helper.DBErr)
			}

			return nil
		}

		if state != WithdrawSuccess && (state != WithdrawFailed && order.Automatic == 0) {
			return errors.New(helper.OrderStateErr)
		}

	case WithdrawAutoPayFailed:
		// 代付失败可以通过手动代付将状态流转至出款中
		if state == WithdrawDealing {
			_, err = meta.MerchantDB.Exec(query)
			if err != nil {
				return pushLog(err, helper.DBErr)
			}

			return nil
		}

		// 代付失败的订单也可以通过手动出款直接将状态流转至出款成功
		// 代付失败的订单还可以通过拒绝直接将状态流转至出款失败
		if state != WithdrawFailed && state != WithdrawSuccess {
			return errors.New(helper.OrderStateErr)
		}

	case WithdrawHangup:
		// 挂起的订单只能领取(该状态流转上传业务已经处理), 该状态只能流转至审核中(WithdrawReviewing)
		return errors.New(helper.OrderStateErr)

	case WithdrawDispatched:
		// 派单状态可流转状态为 挂起(WithdrawHangup) 通过(WithdrawDealing) 拒绝(WithdrawReviewReject)
		// 其中流转至挂起状态由上层业务处理
		if state == WithdrawDealing {
			_, err = meta.MerchantDB.Exec(query)
			if err != nil {
				return pushLog(err, helper.DBErr)
			}

			return nil
		}

		if state != WithdrawReviewReject {
			return errors.New(helper.OrderStateErr)
		}

	default:
		// 审核拒绝, 提款成功, 出款失败三个状态为终态 不能进行其他处理
		return errors.New(helper.OrderStateErr)
	}

	// 3 如果是出款成功,修改订单状态为提款成功,扣除锁定钱包中的钱,发送通知
	if WithdrawSuccess == state {
		return withdrawOrderSuccess(query, bankcard, order)
	}

	order.ReviewRemark = record["review_remark"].(string)
	order.WithdrawRemark = record["withdraw_remark"].(string)
	// 出款失败
	return withdrawOrderFailed(query, order)
}

// 提款成功
func withdrawOrderSuccess(query, bankcard string, order Withdraw) error {

	money := decimal.NewFromFloat(order.Amount)

	// 判断锁定余额是否充足
	_, err := LockBalanceIsEnough(order.UID, money)
	if err != nil {
		return err
	}

	//开启事务
	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	// 更新提款订单状态
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	// 锁定钱包下分
	ex := g.Ex{
		"uid":    order.UID,
		"prefix": meta.Prefix,
	}
	gr := g.Record{
		"last_withdraw_at": order.CreatedAt,
		"lock_amount":      g.L(fmt.Sprintf("lock_amount-%s", money.String())),
	}
	query, _, _ = dialect.Update("tbl_members").Set(gr).Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	err = tx.Commit()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	MemberUpdateCache(order.Username)

	// 修改会员提款限制
	date := time.Unix(order.CreatedAt, 0).Format("20060102")
	_ = withDrawDailyLimitUpdate(money, date, order.Username)

	_ = RocketSendAsync(meta.Prefix+"_finish_flow", []byte(order.ID))
	//提现拒绝就清除稽查缓存
	key := fmt.Sprintf(`%s:check:f:%s`, meta.Prefix, order.Username)
	cmd := meta.MerchantRedis.Unlink(ctx, key)
	fmt.Println("RiskReviewReject", cmd.String(), cmd.Err())

	//发送推送
	msg := fmt.Sprintf(`{"ty":"2","amount": "%f", "ts":"%d","status":"success"}`, order.Amount, order.ConfirmAt)
	fmt.Println(msg)
	topic := fmt.Sprintf("%s/%s/finance", meta.Prefix, order.UID)
	err = Publish(topic, []byte(msg))
	if err != nil {
		fmt.Println("merchantNats.Publish finance = ", err.Error())
		return err
	}

	return nil
}

// 更新每日提款次数限制和金额限制
func withDrawDailyLimitUpdate(amount decimal.Decimal, date, username string) error {

	limitKey := fmt.Sprintf("%s:%s:w:%s", meta.Prefix, username, date)

	var wl = map[string]string{
		"withdraw_count": "0",
		"count_remain":   "0",
		"withdraw_max":   "0.0000",
		"max_remain":     "0.0000",
	}

	// 如果订单生成日的提款限制缓存没有命中 从redis中get key的时 返回的err不会是nil
	// 所以直接返回就可以了
	// 否则需要刷新一下缓存
	rs, err := meta.MerchantRedis.Get(ctx, limitKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return pushLog(err, helper.RedisErr)
	}

	err = helper.JsonUnmarshal([]byte(rs), &wl)
	if err != nil {
		return errors.New(helper.FormatErr)
	}

	count, _ := strconv.ParseInt(wl["count_remain"], 10, 64)
	wl["count_remain"] = strconv.FormatInt(count-1, 10)

	prev, _ := decimal.NewFromString(wl["max_remain"])
	wl["max_remain"] = prev.Sub(amount).String()

	b, _ := helper.JsonMarshal(wl)
	err = meta.MerchantRedis.Set(ctx, limitKey, b, 24*60*60*time.Second).Err()
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}

	return nil
}

// 检查锁定钱包余额是否充足
func LockBalanceIsEnough(uid string, amount decimal.Decimal) (decimal.Decimal, error) {

	balance, err := GetBalanceDB(uid)
	if err != nil {
		return decimal.NewFromFloat(balance.LockAmount), err
	}
	if decimal.NewFromFloat(balance.LockAmount).Sub(amount).IsNegative() {
		return decimal.NewFromFloat(balance.LockAmount), errors.New(helper.LackOfBalance)
	}

	return decimal.NewFromFloat(balance.LockAmount), nil
}

func withdrawOrderFailed(query string, order Withdraw) error {

	money := decimal.NewFromFloat(order.Amount)

	//4、查询用户额度
	balance, err := GetBalanceDB(order.UID)
	if err != nil {
		return err
	}
	balanceAfter := decimal.NewFromFloat(balance.Balance).Add(money)

	//开启事务
	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	//5、更新订单状态
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	//6、更新余额
	ex := g.Ex{
		"uid":    order.UID,
		"prefix": meta.Prefix,
	}
	balanceRecord := g.Record{
		"balance":     g.L(fmt.Sprintf("balance+%s", money.String())),
		"lock_amount": g.L(fmt.Sprintf("lock_amount-%s", money.String())),
	}
	query, _, _ = dialect.Update("tbl_members").Set(balanceRecord).Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	//7、新增账变记录
	mbTrans := memberTransaction{
		AfterAmount:  balanceAfter.String(),
		Amount:       money.String(),
		BeforeAmount: decimal.NewFromFloat(balance.Balance).String(),
		BillNo:       order.ID,
		CreatedAt:    time.Now().UnixNano() / 1e6,
		ID:           helper.GenId(),
		CashType:     helper.TransactionWithDrawFail,
		UID:          order.UID,
		Username:     order.Username,
		Prefix:       meta.Prefix,
	}

	query, _, _ = dialect.Insert("tbl_balance_transaction").Rows(mbTrans).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	err = tx.Commit()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	MemberUpdateCache(order.Username)
	//提现拒绝就清除稽查缓存
	key := fmt.Sprintf(`%s:check:f:%s`, meta.Prefix, order.Username)
	cmd := meta.MerchantRedis.Unlink(ctx, key)
	fmt.Println("RiskReviewReject", cmd.String(), cmd.Err())

	title := "Thông Báo Rút Tiền Thất Bại :"
	content := fmt.Sprintf("Quý Khách Của P3 Thân Mến :\n Đơn Rút Tiền Của Quý Khách Xử Lý Thất Bại, Nguyên Nhân Do : %s. Nếu Có Bất Cứ Vấn Đề Thắc Mắc Vui Lòng Liên Hệ CSKH  Để Biết Thêm Chi Tiết. [P3] Cung Cấp Dịch Vụ Chăm Sóc 1:1 Mọi Lúc Cho Khách Hàng ! \n", order.WithdrawRemark)
	err = messageSend(order.ID, title, content, "system", meta.Prefix, 0, 0, 1, []string{order.Username})
	if err != nil {
		_ = pushLog(err, helper.ESErr)
	}

	//发送推送
	msg := fmt.Sprintf(`{"ty":"2","amount": "%f", "ts":"%d","status":"failed"}`, order.Amount, time.Now().Unix())
	fmt.Println(msg)
	topic := fmt.Sprintf("%s/%s/finance", meta.Prefix, order.UID)

	err = Publish(topic, []byte(msg))
	if err != nil {
		fmt.Println("merchantNats.Publish finance = ", err.Error())
		return err
	}

	return nil
}

// WithdrawHandToAuto 手动代付
func WithdrawHandToAuto(uid, username, id, pid, bid string, amount float64, t time.Time) error {

	bankcard, err := WithdrawGetBank(bid, username)
	if err != nil {
		return err
	}

	// query realName and bankcardNo
	bankcardNo, realName, err := WithdrawGetBkAndRn(bid, uid, false)
	fmt.Println("WithdrawGetBkAndRn realName = ", realName)
	fmt.Println("WithdrawGetBkAndRn bankcardNo = ", bankcardNo)

	if err != nil {

		fmt.Println("WithdrawGetBkAndRn err = ", err)
		return err
	}

	p, err := ChanWithdrawByCateID(pid)
	if err != nil {
		return err
	}

	if len(p.ID) == 0 || p.State == "0" {
		return errors.New(helper.CateNotExist)
	}

	as := strconv.FormatFloat(amount, 'f', -1, 64)
	// check amount range, continue the for loop if amount out of range
	_, ok := validator.CheckFloatScope(as, p.Fmin, p.Fmax)
	if !ok {
		return errors.New(helper.AmountOutRange)
	}

	kvnd := decimal.NewFromInt(1000)
	param := WithdrawAutoParam{
		OrderID:    id,
		Amount:     decimal.NewFromFloat(amount).Mul(kvnd).String(),
		BankID:     bankcard.ID,
		BankCode:   bankCodeMap[bankcard.BankID],
		CardNumber: bankcardNo, // 银行卡号
		CardName:   realName,   // 持卡人姓名
		Ts:         t,          // 时间
		PaymentID:  p.ID,
	}

	// param.BankCode = bank.Code
	oid, err := Withdrawal(vnPay, param)
	if err != nil {
		fmt.Println("withdrawHandToAuto failed 1:", id, err)
		return err
	}

	err = withdrawAutoUpdate(param.OrderID, oid, pid, WithdrawDealing)
	if err != nil {
		fmt.Println("withdrawHandToAuto failed 2:", id, err)
	}

	return nil
}

// 人工出款成功
func WithdrawHandSuccess(id, uid, bid string, record g.Record) error {

	bankcard, err := withdrawGetBankcard(uid, bid)
	if err != nil {
		fmt.Println("query bankcard error: ", err.Error())
	}

	return WithdrawDownPoint(id, bankcard, WithdrawSuccess, record)
}

func withdrawGetBankcard(id, bid string) (string, error) {

	field := "bankcard" + bid
	recs, err := ryrpc.KmsDecryptOne(id, true, []string{field})
	if err != nil {
		_ = pushLog(err, helper.GetRPCErr)
		return "", errors.New(helper.GetRPCErr)
	}

	return recs[field], nil
}

func WithdrawGetBkAndRn(bid, uid string, hide bool) (string, string, error) {

	field := "bankcard" + bid
	recs, err := ryrpc.KmsDecryptOne(uid, hide, []string{"realname", field})
	if err != nil {
		_ = pushLog(err, helper.GetRPCErr)
		return "", "", errors.New(helper.GetRPCErr)
	}

	return recs[field], recs["realname"], nil
}

func ChanWithdrawByCateID(cid string) (Payment_t, error) {

	var channel Payment_t

	ex := g.Ex{
		"cate_id":    cid,
		"channel_id": []int{7, 101},
	}
	query, _, _ := dialect.From("f2_payment").Select(colPayment...).Where(ex).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&channel, query)
	if err != nil && err != sql.ErrNoRows {
		return channel, pushLog(err, helper.DBErr)
	}

	return channel, nil
}

// Withdrawal 提现
func Withdrawal(p Payment, arg WithdrawAutoParam) (string, error) {

	// 维护订单 渠道信息
	ex := g.Ex{
		"id": arg.OrderID,
	}
	record := g.Record{
		"pid": arg.PaymentID,
	}
	err := withdrawUpdateInfo(ex, record)
	if err != nil {
		return "", pushLog(err, helper.DBErr)
	}

	data, err := p.Withdraw(arg)
	if err != nil {
		if err.Error() == "no enough money" {
			return "", errors.New("该通道余额不足，请尝试其他通道")
		} else {
			return "", errors.New(helper.ChannelBusyTryOthers)
		}
	}

	return data.OrderID, nil
}

func withdrawUpdateInfo(ex g.Ex, record g.Record) error {

	ex["prefix"] = meta.Prefix
	query, _, _ := dialect.Update("tbl_withdraw").Set(record).Where(ex).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	return err
}

func withdrawAutoUpdate(id, oid, pid string, state int) error {

	r := g.Record{"state": state, "automatic": "1"}
	if oid != "" {
		r["oid"] = oid
	}
	if pid != "" {
		r["pid"] = pid
	}

	query, _, _ := dialect.Update("tbl_withdraw").Set(r).Where(g.Ex{"id": id}).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return err
}

// WithdrawAutoPaySetFailed 将订单状态从出款中修改为代付失败
func WithdrawAutoPaySetFailed(id string, confirmAt int64, confirmUid, confirmName string) error {

	order, err := WithdrawFind(id)
	if err != nil {
		return err
	}

	// 只能将出款中(三方处理中, 即自动代付)的订单状态流转为代付失败
	if order.State != WithdrawDealing || order.Automatic != 1 {
		return errors.New(helper.OrderStateErr)
	}

	// 将automatic设为1是为了确保状态为代付失败的订单一定为自动出款(automatic=1)
	record := g.Record{
		"state":     WithdrawAutoPayFailed,
		"automatic": "1",
	}

	return WithdrawDownPoint(id, "", WithdrawAutoPayFailed, record)
}

// 每日剩余提款次数和总额
func WithdrawLimit(ctx *fasthttp.RequestCtx) (map[string]string, error) {

	member, err := MemberCache(ctx)
	if err != nil {
		return nil, err
	}

	date := ctx.Time().Format("20060102")
	return WithDrawDailyLimit(date, member.Username)
}

// WithDrawDailyLimit 获取每日提现限制
func WithDrawDailyLimit(date, username string) (map[string]string, error) {

	limitKey := fmt.Sprintf("%s:%s:w:%s", meta.Prefix, username, date)

	// 获取会员当日提现限制，先取缓存 没有就设置一下
	num, err := meta.MerchantRedis.Exists(ctx, limitKey).Result()
	if err != nil {
		return defaultLevelWithdrawLimit, pushLog(err, helper.RedisErr)
	}

	if num > 0 {
		rs, err := meta.MerchantRedis.Get(ctx, limitKey).Result()
		if err != nil {
			return defaultLevelWithdrawLimit, pushLog(err, helper.RedisErr)
		}

		data := make(map[string]string)
		err = helper.JsonUnmarshal([]byte(rs), &data)
		if err != nil {
			return defaultLevelWithdrawLimit, errors.New(helper.FormatErr)
		}

		return data, nil
	}

	b, _ := helper.JsonMarshal(defaultLevelWithdrawLimit)
	err = meta.MerchantRedis.Set(ctx, limitKey, b, 24*60*60*time.Second).Err()
	if err != nil {
		return defaultLevelWithdrawLimit, pushLog(err, helper.RedisErr)
	}

	return defaultLevelWithdrawLimit, nil
}

func WithdrawInProcessing(ctx *fasthttp.RequestCtx) (map[string]interface{}, error) {

	data := map[string]interface{}{}

	member, err := MemberCache(ctx)
	if err != nil {
		return data, err
	}
	cd, err := ConfigDetail()
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}
	//存款订单配置开启
	dts := cd["withdraw_min"]
	data["min_amount"] = dts
	order := Withdraw{}
	ex := g.Ex{
		"uid":   member.UID,
		"state": g.Op{"notIn": []int64{WithdrawReviewReject, WithdrawSuccess, WithdrawFailed}},
	}
	query, _, _ := dialect.From("tbl_withdraw").Select(colsWithdraw...).Where(ex).Limit(1).ToSQL()
	err = meta.MerchantDB.Get(&order, query)
	if err != nil && err != sql.ErrNoRows {
		return data, pushLog(err, helper.DBErr)
	}

	if err == sql.ErrNoRows {
		return data, nil
	}

	data = map[string]interface{}{
		"id":          order.ID,
		"bid":         order.BID,
		"amount":      order.Amount,
		"state":       order.State,
		"created_at":  order.CreatedAt,
		"rate":        order.VirtualRate,
		"count":       order.VirtualCount,
		"wallet_addr": order.WalletAddr,
		"min_amount":  dts,
	}

	return data, nil
}

func WithdrawAuto(param WithdrawAutoParam, level int) error {

	i := 0
	key := fmt.Sprintf("%s:pw:%d", meta.Prefix, level)
	pwc, err := meta.MerchantRedis.LLen(ctx, key).Result()
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}

	for {
		i++

		// the maximum loop times is the length of the list
		if pwc < int64(i) {
			fmt.Printf("withdrawAuto failed 1: %v \n", param)
			return errors.New(helper.NoPayChannel)
		}

		info, err := ChanByID("133221087319615487")
		if err != nil {
			continue
		}

		// amount must be divided by 1000, because the unit of fmin and fmax is k
		amount, _ := decimal.NewFromString(param.Amount)
		amount = amount.Div(decimal.NewFromInt(1000))

		// check amount range, continue the for loop if amount out of range
		_, ok := validator.CheckFloatScope(amount.String(), info.Fmin, info.Fmax)
		if !ok {
			fmt.Println("withdrawAuto failed 4:", param, amount.String(), info.Fmin, info.Fmax)
			continue
		}

		param.PaymentID = "133221087319615487"
		oid, err := Withdrawal(vnPay, param)
		if err != nil {
			fmt.Println("withdrawAuto failed 6:", param, err)
			return err
		}

		_ = withdrawAutoUpdate(param.OrderID, oid, param.PaymentID, WithdrawDealing)
		return nil
	}
}

func WithdrawRiskReview(id string, state int, record g.Record, withdraw Withdraw) error {

	err := WithdrawDownPoint(id, "", state, record)
	if err != nil {
		return err
	}

	var confirmUID string
	ex := g.Ex{
		"id": id,
	}
	query, _, _ := dialect.From("tbl_withdraw").Select("confirm_uid").Where(ex).Limit(1).ToSQL()
	err = meta.MerchantDB.Get(&confirmUID, query)
	if err != nil {
		fmt.Println(pushLog(err, helper.DBErr))
	}

	if confirmUID != "" && confirmUID != withdraw.ConfirmUID {
		_ = SetRisksOrder(confirmUID, id, -1)
	}

	_ = SetRisksOrder(withdraw.ConfirmUID, id, -1)

	return nil
}

func WithdrawApplyListData(data FWithdrawData) (WithdrawListData, error) {

	result := WithdrawListData{
		T:   data.T,
		Agg: data.Agg,
	}

	if len(data.D) == 0 {
		return result, nil
	}

	// 获取渠道号的pid slice
	pids := make([]string, 0)
	var agencyNames []string
	// 组装获取rpc数据参数
	rpcParam := make(map[string][]string)
	namesMap := make(map[string]string)
	for _, v := range data.D {
		rpcParam["bankcard"] = append(rpcParam["bankcard"], v.BID)
		rpcParam["realname"] = append(rpcParam["realname"], v.UID)
		namesMap[v.Username] = v.UID
		pids = append(pids, v.PID)

		if v.ParentName != "" && v.ParentName != "root" {
			agencyNames = append(agencyNames, v.ParentName)
		}

	}
	userMap := map[string]MBBalance{}

	var uids []string

	if len(data.D) > 0 {

		for _, v := range data.D {
			uids = append(uids, v.UID)
		}

		balances, err := getBalanceByUids(uids)
		if err != nil {
			return result, err
		}

		for _, v := range balances {
			userMap[v.UID] = v
		}
	}
	wm, err := withdrawFirst(uids)
	if err != nil {
		return result, err
	}
	// 遍历用户map 读取标签数据
	var names []string
	tags := make(map[string]string)
	for name, uid := range namesMap {
		// 获取用户标签
		memberTag, err := MemberTagsList(uid)
		if err != nil {
			return result, err
		}
		// 组装需要通过name获取的 redis参数
		names = append(names, name)
		tags[name] = memberTag
	}

	bankcards, err := bankcardListDBByIDs(rpcParam["bankcard"])
	if err != nil {
		return result, err
	}

	encFields := []string{"realname"}

	for _, v := range rpcParam["bankcard"] {
		encFields = append(encFields, "bankcard"+v)
	}

	recs, err := ryrpc.KmsDecryptAll(rpcParam["realname"], false, encFields)
	if err != nil {
		_ = pushLog(err, helper.GetRPCErr)
		return result, errors.New(helper.GetRPCErr)
	}

	cids, _ := channelCateMap(pids)

	// 处理返回前端的数据
	for _, v := range data.D {

		//fmt.Println("k = ", k)

		var temp []MemberDepositInfo
		ex := g.Ex{"uid": v.UID, "deposit_at": g.Op{"lt": v.CreatedAt}}
		query, _, _ := dialect.From("tbl_member_deposit_info").Select(g.C("uid"), g.C("deposit_amount"),
			g.C("deposit_at"), g.C("prefix"), g.C("flags")).Where(ex).Order(g.C("flags").Desc()).Limit(1).ToSQL()
		err = meta.MerchantDB.Select(&temp, query)
		if err != nil {
			return result, pushLog(err, helper.DBErr)
		}

		wat := wm[v.UID]
		w := withdrawCols{
			mWithdraw:          v,
			MemberBankNo:       recs[v.UID]["bankcard"+v.BID],
			MemberBankRealName: recs[v.UID]["realname"],
			MemberRealName:     recs[v.UID]["realname"],
			MemberTags:         tags[v.Username],
			Balance:            v.Balance,
			LockAmount:         userMap[v.UID].LockAmount,
		}
		if wat > 0 && wat != v.CreatedAt {
			w.FirstWithdraw = false
		} else if wat == v.CreatedAt || wat == 0 {
			w.FirstWithdraw = true
		}
		if len(temp) > 0 {
			w.LastDepositAt = temp[0].DepositAt
			w.LastDepositAmount = temp[0].DepositAmount
		}

		// 匹配银行卡信息
		card, ok := bankcards[v.BID]
		if ok {
			w.MemberBankName = card.BankID
			w.MemberBankAddress = card.BankAddress
		}

		// 匹配渠道信息
		cate, ok := cids[v.PID]
		if ok {
			w.CateID = cate.ID
			w.CateName = cate.Name
		}

		result.D = append(result.D, w)
	}

	return result, nil
}

// WithdrawalCallBack 提款回调
func WithdrawalCallBack(fctx *fasthttp.RequestCtx) {

	var (
		err  error
		data paymentCallbackResp
	)

	// 获取并校验回调参数
	data, err = vnPay.WithdrawCallBack(fctx)
	if err != nil {
		fctx.SetBody([]byte(`failed`))
		pushLog(err, helper.WithdrawFailure)
		return
	}
	fmt.Println("获取并校验回调参数:", data)

	// 查询订单
	order, err := withdrawFind(data.OrderID)
	if err != nil {
		err = fmt.Errorf("query order error: [%v]", err)
		fctx.SetBody([]byte(`failed`))
		pushLog(err, helper.WithdrawFailure)
		return
	}

	// 提款成功只考虑出款中和代付失败的情况
	// 审核中的状态不用考虑，因为不会走到三方去，出款成功和出款失败是终态也不用考虑
	if order.State != WithdrawDealing && order.State != WithdrawAutoPayFailed {
		err = fmt.Errorf("duplicated Withdrawal notify: [%v]", err)
		fctx.SetBody([]byte(`failed`))
		pushLog(err, helper.WithdrawFailure)
		return
	}

	if data.Amount != "-1" {
		// 校验money, 暂时不处理订单与最初订单不一致的情况
		// 兼容越南盾的单位K 与 人民币元
		if data.Cent == 0 {
			data.Cent = 1000
		}
		err = compareAmount(data.Amount, fmt.Sprintf("%.4f", order.Amount), data.Cent)
		if err != nil {
			err = fmt.Errorf("compare amount error: [%v]", err)
			fctx.SetBody([]byte(`failed`))
			pushLog(err, helper.WithdrawFailure)
			return
		}
	}
	now := fctx.Time()
	if data.PayAt != "" && data.PayAt != "0" {
		confirmAt, err := strconv.ParseInt(data.PayAt, 10, 64)
		if err == nil {
			if len(data.PayAt) == 13 {
				confirmAt = confirmAt / 1000
			}
			now = time.Unix(confirmAt, 0)
		}
	}
	// 修改订单状态
	err = withdrawUpdate(data.OrderID, order.UID, order.BID, data.State, now)
	if err != nil {
		err = fmt.Errorf("set order state [%d] to [%d] error: [%v]", order.State, data.State, err)
		pushLog(err, helper.WithdrawFailure)
		fctx.SetBody([]byte(`failed`))
		return
	}

	if data.Resp != nil {
		fctx.SetStatusCode(200)
		fctx.SetContentType("application/json")
		bytes, err := helper.JsonMarshal(data.Resp)
		if err != nil {
			fctx.SetBody([]byte(err.Error()))
			return
		}
		fctx.SetBody(bytes)
		return
	}

	fctx.SetBody([]byte(`success`))
}

// 查找单条提款记录, 订单不存在返回错误: OrderNotExist
func withdrawFind(id string) (Withdraw, error) {

	w := Withdraw{}
	query, _, _ := dialect.From("tbl_withdraw").Select(colsWithdraw...).Where(g.Ex{"id": id}).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&w, query)
	if err == sql.ErrNoRows {
		return w, errors.New(helper.OrderNotExist)
	}

	if err != nil {
		return w, err
	}

	return w, nil
}

// 接收到三方回调后调用这个方法（三方调用缺少confirm uid和confirm name这些信息）
func withdrawUpdate(id, uid, bid string, state int, t time.Time) error {

	// 加锁
	err := withdrawLock(id)
	if err != nil {
		return err
	}
	defer withdrawUnLock(id)

	record := g.Record{
		"state": state,
	}

	switch state {
	case WithdrawSuccess:
		record["automatic"] = "1"
		record["withdraw_at"] = fmt.Sprintf("%d", t.Unix())
	case WithdrawAutoPayFailed:
		record["confirm_at"] = fmt.Sprintf("%d", t.Unix())
	default:
		return errors.New(helper.StateParamErr)
	}

	bankcard, err := withdrawGetBankcard(uid, bid)
	if err != nil {
		fmt.Println("query bankcard error: ", err.Error())
	}

	return WithdrawDownPoint(id, bankcard, state, record)
}

// WithdrawLock 锁定提款订单
// 订单因为外部因素(接口)导致的状态流转应该加锁
func withdrawLock(id string) error {

	key := fmt.Sprintf(withdrawOrderLockKey, id)
	return Lock(key)
}

// WithdrawUnLock 解锁提款订单
func withdrawUnLock(id string) {

	key := fmt.Sprintf(withdrawOrderLockKey, id)
	Unlock(key)
}
