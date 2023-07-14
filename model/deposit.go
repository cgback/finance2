package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/go-redis/redis/v8"
	"github.com/shopspring/decimal"
	"github.com/tenfyzhong/cityhash"
	"github.com/valyala/fasthttp"
	"strconv"
	"time"
)

// 获取用户上笔订单存款金额
func depositLastAmount(uid string) (float64, error) {

	ex := g.Ex{
		"uid":    uid,
		"state":  DepositSuccess,
		"amount": g.Op{"gt": 0},
	}
	var amount float64
	query, _, _ := dialect.From("tbl_deposit").Select("amount").Where(ex).Order(g.I("created_at").Desc()).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&amount, query)
	if err != nil && err != sql.ErrNoRows {
		return amount, pushLog(err, helper.DBErr)
	}

	return amount, nil
}

// 存入数据库
func deposit(record g.Record) error {

	query, _, _ := dialect.Insert("tbl_deposit").Rows(record).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	fmt.Println("deposit insert:", query)
	if err != nil {
		return err

	}

	return nil
}

func DepositFindOne(id string) (Deposit, error) {

	ex := g.Ex{
		"id": id,
	}

	return DepositOrderFindOne(ex)
}

// DepositOrderFindOne 查询存款订单
func DepositOrderFindOne(ex g.Ex) (Deposit, error) {

	ex["prefix"] = meta.Prefix
	order := Deposit{}
	query, _, _ := dialect.From("tbl_deposit").Select(colsDeposit...).Where(ex).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&order, query)
	if err == sql.ErrNoRows {
		return order, errors.New(helper.OrderNotExist)
	}

	if err != nil {
		return order, pushLog(err, helper.DBErr)
	}

	return order, nil
}

// DepositRecordUpdate 更新订单信息
func DepositRecordUpdate(id string, record g.Record) error {

	ex := g.Ex{
		"id": id,
	}
	toSQL, _, _ := dialect.Update("tbl_deposit").Where(ex).Set(record).ToSQL()
	_, err := meta.MerchantDB.Exec(toSQL)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return nil
}

func DepositUpPointReviewCancel(did, uid, name, remark string, state int) error {

	// 判断状态是否合法
	allow := map[int]bool{
		DepositCancelled: true,
		DepositSuccess:   true,
	}
	if _, ok := allow[state]; !ok {
		return errors.New(helper.OrderStateErr)
	}

	// 判断订单是否存在
	ex := g.Ex{"id": did, "state": []int{DepositReviewing, DepositConfirming}}
	order, err := DepositOrderFindOne(ex)
	if err != nil {
		return err
	}

	now := time.Now()

	record := g.Record{
		"state":         state,
		"confirm_at":    now.Unix(),
		"confirm_uid":   uid,
		"confirm_name":  name,
		"review_remark": remark,
	}
	query, _, _ := dialect.Update("tbl_deposit").Set(record).Where(ex).ToSQL()
	_, err = meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	//发送站内信
	title := "Thông Báo Nạp Tiền Thất Bại:"
	content := fmt.Sprintf(" Quý Khách Của P3 Thân Mến :\n Đơn Nạp Tiền Của Quý Khách Xử Lý Thất Bại, Nguyên Nhân Do : %s. Nếu Có Bất Cứ Vấn Đề Thắc Mắc Vui Lòng Liên Hệ CSKH  Để Biết Thêm Chi Tiết. [P3] Cung Cấp Dịch Vụ Chăm Sóc 1:1 Mọi Lúc Cho Khách Hàng ! \n", remark)
	err = messageSend(order.ID, title, content, "system", meta.Prefix, 0, 0, 1, []string{order.Username})
	if err != nil {
		_ = pushLog(err, helper.ESErr)
	}

	//发送推送
	msg := fmt.Sprintf(`{"ty":"1","amount": "%f", "ts":"%d","status":"failed"}`, order.Amount, time.Now().Unix())
	fmt.Println(msg)
	topic := fmt.Sprintf("%s/%s/finance", meta.Prefix, order.UID)

	err = Publish(topic, []byte(msg))
	if err != nil {
		fmt.Println("merchantNats.Publish finance = ", err.Error())
		return err
	}

	return nil
}

func DepositUpPointReviewSuccess(did, uid, name, remark string, state int) error {

	// 判断状态是否合法
	allow := map[int]bool{
		DepositCancelled: true,
		DepositSuccess:   true,
	}
	if _, ok := allow[state]; !ok {
		return errors.New(helper.OrderStateErr)
	}

	// 判断订单是否存在
	ex := g.Ex{"id": did, "state": []int{DepositReviewing, DepositConfirming}}
	order, err := DepositOrderFindOne(ex)
	if err != nil {
		return err
	}

	now := time.Now()
	money := decimal.NewFromFloat(order.Amount)
	amount := money.String()
	// 后面都是存款成功 和 下分失败 的处理
	// 1、查询用户额度
	balance, err := GetBalanceDB(order.UID)
	if err != nil {
		return err
	}
	balanceAfter := decimal.NewFromFloat(balance.Balance).Add(money)

	balanceFeeAfter := balanceAfter
	fee := decimal.Zero
	var feeCashType int
	//如果存款有优惠

	key := meta.Prefix + ":f:p:" + order.PID

	promoDiscount, err := meta.MerchantRedis.HGet(ctx, key, "discount").Result()
	if err != nil && err != redis.Nil {
		//缓存没有配置就跳过
		fmt.Println(err)
	}
	fmt.Println("review discount:", promoDiscount)
	pd, _ := decimal.NewFromString(promoDiscount)
	if pd.GreaterThan(decimal.Zero) {
		//大于0就是优惠，给钱
		fee = money.Mul(pd).Div(decimal.NewFromInt(100))
		money = money.Add(fee)
		balanceFeeAfter = decimal.NewFromFloat(balance.Balance).Add(money.Abs())
		feeCashType = helper.TransactionDepositBonus
	} else if pd.LessThan(decimal.Zero) {
		//小于0就是收费，扣钱
		fee = money.Mul(pd).Div(decimal.NewFromInt(100))
		money = money.Add(fee)
		balanceFeeAfter = decimal.NewFromFloat(balance.Balance).Add(money.Abs())
		feeCashType = helper.TransactionDepositFee
	}

	// 开启事务
	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	// 2、更新订单状态
	record := g.Record{
		"state":         state,
		"confirm_at":    now.Unix(),
		"confirm_uid":   uid,
		"confirm_name":  name,
		"review_remark": remark,
	}
	record["discount"] = fee

	query, _, _ := dialect.Update("tbl_deposit").Set(record).Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	// 3、更新余额
	ex = g.Ex{
		"uid":    order.UID,
		"prefix": meta.Prefix,
	}
	br := g.Record{
		"balance": g.L(fmt.Sprintf("balance+%s", money.String())),
	}
	query, _, _ = dialect.Update("tbl_members").Set(br).Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	// 4、新增账变记录
	id := helper.GenId()
	mbTrans := memberTransaction{
		AfterAmount:  balanceAfter.String(),
		Amount:       amount,
		BeforeAmount: decimal.NewFromFloat(balance.Balance).String(),
		BillNo:       order.ID,
		CreatedAt:    now.UnixMilli(),
		ID:           id,
		CashType:     helper.TransactionDeposit,
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

	if balanceFeeAfter.Cmp(balanceAfter) != 0 {
		//手续费/优惠的帐变
		id = helper.GenId()
		mbTrans = memberTransaction{
			AfterAmount:  balanceFeeAfter.String(),
			Amount:       fee.String(),
			BeforeAmount: balanceAfter.String(),
			BillNo:       order.ID,
			CreatedAt:    time.Now().UnixMilli(),
			ID:           id,
			CashType:     feeCashType,
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
	}

	err = tx.Commit()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}
	defer MemberUpdateCache(order.Username)

	rec := g.Record{
		"first_deposit_at":     order.CreatedAt,
		"first_deposit_amount": order.Amount,
	}
	ex1 := g.Ex{
		"username":         order.Username,
		"first_deposit_at": 0,
	}
	query, _, _ = dialect.Update("tbl_members").Set(rec).Where(ex1).ToSQL()
	fmt.Printf("memberFirstDeposit Update: %v\n", query)
	result, err := meta.MerchantDB.Exec(query)
	if err != nil {
		fmt.Println("update member first_amount err:", err.Error())
	}

	updateRows, _ := result.RowsAffected()
	if updateRows == 0 {
		rec = g.Record{
			"second_deposit_at":     order.CreatedAt,
			"second_deposit_amount": order.Amount,
		}
		ex = g.Ex{
			"username":          order.Username,
			"second_deposit_at": 0,
		}
		query, _, _ = dialect.Update("tbl_members").Set(rec).Where(ex).ToSQL()
		fmt.Printf("memberSecondDeposit Update: %v\n", query)
		_, err = meta.MerchantDB.Exec(query)
		if err != nil {
			fmt.Println("update member second_amount err:", err.Error())
		}
	}

	_ = RocketSendAsync(meta.Prefix+"_finish_flow", []byte("D_"+order.ID))

	//发送推送
	msg := fmt.Sprintf(`{"ty":"1","amount": "%f", "ts":"%d","status":"success"}`, order.Amount, time.Now().Unix())
	fmt.Println(msg)
	topic := fmt.Sprintf("%s/%s/finance", meta.Prefix, order.UID)

	err = Publish(topic, []byte(msg))
	if err != nil {
		fmt.Println("merchantNats.Publish finance = ", err.Error())
		return err
	}

	lastDepositKey := fmt.Sprintf("%s:uld:%s", meta.Prefix, order.Username)
	_ = meta.MerchantRedis.Set(ctx, lastDepositKey, order.ChannelID, 100*time.Hour).Err()
	lastDepositPaymentKey := fmt.Sprintf("%s:uldp:%s", meta.Prefix, order.Username)
	_ = meta.MerchantRedis.Set(ctx, lastDepositPaymentKey, order.PID, 100*time.Hour).Err()

	return nil
}

// 获取用户上笔订单存款
func depositLast(uid string) (Deposit, error) {

	ex := g.Ex{
		"uid":    uid,
		"state":  DepositSuccess,
		"amount": g.Op{"gt": 0},
	}
	var order Deposit
	query, _, _ := dialect.From("tbl_deposit").Select(colsDeposit...).Where(ex).Order(g.I("confirm_at").Desc()).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&order, query)
	if err != nil && err != sql.ErrNoRows {
		return order, pushLog(err, helper.DBErr)
	}

	return order, nil
}

func DepositDetail(username, state, channelID, timeFlag, startTime, endTime string, page, pageSize int) (FDepositData, error) {

	data := FDepositData{}
	ex := g.Ex{
		"amount": g.Op{"gt": 0.00},
		"prefix": meta.Prefix,
	}

	if username != "" {
		ex["username"] = username
	}

	if channelID != "" {
		ex["channel_id"] = channelID
	}

	if state != "" {
		ex["state"] = state
	}

	order := "created_at"
	if startTime != "" && endTime != "" {

		startAt, err := helper.TimeToLoc(startTime, loc)
		if err != nil {
			return data, errors.New(helper.DateTimeErr)
		}

		endAt, err := helper.TimeToLoc(endTime, loc)
		if err != nil {
			return data, errors.New(helper.DateTimeErr)
		}

		if timeFlag == "2" {
			order = "confirm_at"
			ex["confirm_at"] = g.Op{"between": exp.NewRangeVal(startAt, endAt)}
		} else {
			ex["created_at"] = g.Op{"between": exp.NewRangeVal(startAt, endAt)}
		}
	}

	if page == 1 {

		var total dataTotal
		query, _, _ := dialect.From("tbl_deposit").Select(g.COUNT(1).As("t"), g.SUM("amount").As("s")).Where(ex).ToSQL()
		err := meta.MerchantDB.Get(&total, query)
		if err != nil {
			return data, pushLog(err, helper.DBErr)
		}

		if total.T.Int64 < 1 {
			return data, nil
		}

		data.Agg = map[string]string{
			"amount": fmt.Sprintf("%.4f", total.S.Float64),
		}

		// 查询到账金额和上分金额 (当前需求到账金额和上分金额用一个字段)
		exc := g.Ex{
			"prefix": meta.Prefix,
		}
		for k, v := range ex {
			exc[k] = v
		}
		exc["state"] = DepositSuccess
		query, _, _ = dialect.From("tbl_deposit").Select(g.COALESCE(g.SUM("amount"), 0).As("s")).Where(exc).ToSQL()
		err = meta.MerchantDB.Get(&total, query)
		if err != nil {
			return data, pushLog(err, helper.DBErr)
		}

		data.Agg["valid_amount"] = fmt.Sprintf("%.4f", total.S.Float64)
		data.T = total.T.Int64
	}

	offset := uint((page - 1) * pageSize)
	query, _, _ := dialect.From("tbl_deposit").Select(colsDeposit...).
		Where(ex).Offset(offset).Limit(uint(pageSize)).Order(g.C(order).Desc()).ToSQL()
	err := meta.MerchantDB.Select(&data.D, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	return data, nil
}

// DepositReview 存款补单审核
func DepositReview(did, remark, state, name, uid string) error {

	// 加锁
	err := depositLock(did)
	if err != nil {
		return err
	}
	defer depositUnLock(did)

	iState, _ := strconv.Atoi(state)

	ex := g.Ex{"id": did, "state": DepositConfirming}
	order, err := DepositOrderFindOne(ex)
	if err != nil {
		return err
	}

	// 充值成功处理订单状态
	if iState == DepositSuccess {
		_ = CacheDepositProcessingRem(order.UID)
		err = DepositUpPointReviewSuccess(did, uid, name, remark, iState)
		if err != nil {
			return err
		}
	} else {
		err = DepositUpPointReviewCancel(did, uid, name, remark, iState)
		if err != nil {
			return err
		}
	}

	return nil
}

// DepositHistory 存款历史列表
func DepositHistory(username, parentName, groupName, id, channelID, oid, state,
	minAmount, maxAmount, startTime, endTime, cid, sortField string, timeFlag uint8, flag, page, pageSize, ty, dty, isAsc int) (FDepositData, error) {

	data := FDepositData{}

	startAt, err := helper.TimeToLoc(startTime, loc)
	if err != nil {
		return data, errors.New(helper.DateTimeErr)
	}

	endAt, err := helper.TimeToLoc(endTime, loc)
	if err != nil {
		return data, errors.New(helper.DateTimeErr)
	}
	ex := g.Ex{
		"prefix": meta.Prefix,
		"tester": 1,
	}
	if username != "" {
		ex["username"] = username
	}
	if parentName != "" {
		ex["parent_name"] = parentName
	}
	if groupName != "" {
		topName, err := TopNameByGroup(groupName)
		if err != nil {
			pushLog(err, helper.DBErr)
		}
		if topName == "" {
			return data, errors.New(helper.UsernameErr)
		}
		ex["top_name"] = topName
	}
	if dty > 0 {
		ex["success_time"] = dty
	}

	if timeFlag == 1 {
		ex["created_at"] = g.Op{"between": exp.NewRangeVal(startAt, endAt)}
	} else {
		ex["confirm_at"] = g.Op{"between": exp.NewRangeVal(startAt, endAt)}
	}

	if id != "" {
		ex["id"] = id
	}

	if channelID != "" {
		ex["channel_id"] = channelID
	}

	if oid != "" {
		ex["oid"] = oid
	}

	if cid != "" {
		ex["cid"] = cid
	}

	if state != "" && state != "0" {
		ex["state"] = state
	} else {
		ex["state"] = []int{DepositSuccess, DepositCancelled}
	}

	if ty != 0 {
		ex["flag"] = ty
	}

	ex["amount"] = g.Op{"gt": 0}
	// 下分列表
	if flag == 1 {
		ex["amount"] = g.Op{"lt": 0}
	}
	if minAmount != "" && maxAmount != "" {
		minF, err := strconv.ParseFloat(minAmount, 64)
		if err != nil {
			return data, pushLog(err, helper.AmountErr)
		}

		maxF, err := strconv.ParseFloat(maxAmount, 64)
		if err != nil {
			return data, pushLog(err, helper.AmountErr)
		}

		ex["amount"] = g.Op{"between": exp.NewRangeVal(minF, maxF)}

	}

	if page == 1 {

		var total dataTotal
		query, _, _ := dialect.From("tbl_deposit").Select(g.COUNT(1).As("t"), g.SUM("amount").As("s")).Where(ex).ToSQL()
		err = meta.MerchantDB.Get(&total, query)
		if err != nil {
			return data, pushLog(err, helper.DBErr)
		}

		if total.T.Int64 < 1 {
			return data, nil
		}

		data.Agg = map[string]string{
			"amount": fmt.Sprintf("%.4f", total.S.Float64),
		}

		data.T = total.T.Int64
	}

	offset := uint((page - 1) * pageSize)
	orderField := g.L("username")
	if sortField != "" {
		orderField = g.L(sortField)
	} else {
		if timeFlag == 1 {
			orderField = g.L("created_at")
		} else {
			orderField = g.L("confirm_at")
		}
	}

	orderBy := orderField.Desc()
	if isAsc == 1 {
		orderBy = orderField.Asc()
	}
	query, _, _ := dialect.From("tbl_deposit").Select(colsDeposit...).
		Where(ex).Offset(offset).Limit(uint(pageSize)).Order(orderBy).ToSQL()
	err = meta.MerchantDB.Select(&data.D, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	for i := 0; i < len(data.D); i++ {
		if data.D[i].ReviewRemark == "undefined" {
			data.D[i].ReviewRemark = ""
		}
		mb, err := GetMemberCache(data.D[i].Username)
		if err != nil {
			return data, pushLog(err, helper.RedisErr)
		}
		data.D[i].GroupName = mb.GroupName
	}

	return data, nil
}

// DepositList 存款订单列表
func DepositList(ex g.Ex, startTime, endTime string, isBig, firstWd, page, pageSize int) (FDepositData, error) {

	ex["prefix"] = meta.Prefix
	ex["tester"] = 1

	data := FDepositData{}

	if startTime != "" && endTime != "" {

		startAt, err := helper.TimeToLoc(startTime, loc)
		if err != nil {
			return data, errors.New(helper.DateTimeErr)
		}

		endAt, err := helper.TimeToLoc(endTime, loc)
		if err != nil {
			return data, errors.New(helper.DateTimeErr)
		}

		ex["created_at"] = g.Op{"between": exp.NewRangeVal(startAt, endAt)}
	}

	if page == 1 {

		total := dataTotal{}
		countQuery, _, _ := dialect.From("tbl_deposit").Select(g.COUNT(1).As("t"), g.SUM("amount").As("s")).Where(ex).ToSQL()
		err := meta.MerchantDB.Get(&total, countQuery)
		if err != nil {
			return data, pushLog(err, helper.DBErr)
		}

		if total.T.Int64 < 1 {
			return data, nil
		}

		data.Agg = map[string]string{
			"amount": fmt.Sprintf("%.4f", total.S.Float64),
		}
		data.T = total.T.Int64
	}

	offset := uint((page - 1) * pageSize)
	query, _, _ := dialect.From("tbl_deposit").Select(colsDeposit...).
		Where(ex).Offset(offset).Limit(uint(pageSize)).Order(g.C("created_at").Desc()).ToSQL()

	err := meta.MerchantDB.Select(&data.D, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	var ids []string
	for _, v := range data.D {
		ids = append(ids, v.UID)
	}

	return data, nil
}

// DepositUSDTReview 线下USDT-存款审核
func DepositUSDTReview(did, remark, name, adminUID, depositUID string, state int) error {

	// 加锁
	err := depositLock(did)
	if err != nil {
		return err
	}
	defer depositUnLock(did)

	// 充值成功处理订单状态
	if state == DepositSuccess {
		err = DepositUpPointReviewSuccess(did, adminUID, name, remark, state)
		if err != nil {
			return err
		}
		_ = CacheDepositProcessingRem(depositUID)
	} else {
		err = DepositUpPointReviewCancel(did, adminUID, name, remark, state)
		if err != nil {
			return err
		}
	}

	contentLog := fmt.Sprintf("财务管理-存款管理-USDT存款-线下审核订单-%s:后台账号:%s【订单号:%s,】", StateDesMap[state], name, did)
	AdminLogInsert(DepositModel, contentLog, OpDesMap[state], name)
	return nil
}

// DepositManual 手动补单
func DepositManual(id, amount, remark, name, uid string) error {

	money, _ := decimal.NewFromString(amount)
	if money.Cmp(zero) < 1 {
		return errors.New(helper.AmountErr)
	}
	// 判断订单是否存在
	oEx := g.Ex{"id": id, "automatic": 1}
	order, err := DepositOrderFindOne(oEx)
	if err != nil {
		return err
	}

	// 判断状态
	if order.State != DepositConfirming {
		return errors.New(helper.OrderStateErr)
	}

	// 判断此订单是否已经已经有一笔补单成功,如果这笔订单的手动补单有一笔成功,则不允许再补单
	existEx := g.Ex{
		"oid":       order.OID,
		"state":     DepositSuccess,
		"automatic": 0,
		"prefix":    meta.Prefix,
	}
	_, err = DepositOrderFindOne(existEx)
	if err != nil && err.Error() == helper.DBErr {
		return err
	}
	if err == nil {
		return errors.New(helper.OrderExist)
	}

	var tester string
	mb, err := GetMemberCache(order.Username)
	if err != nil {
		return errors.New(helper.UserNotExist)
	}
	tester = mb.Tester

	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return errors.New(helper.TransErr)
	}

	now := time.Now()
	// 生成订单
	newid := helper.GenId()
	ca := now.Unix()
	sn := fmt.Sprintf(`deposit%s%s%d%d`, newid, order.Username, ca, mb.CreatedAt)
	mhash := fmt.Sprintf("%d", cityhash.CityHash64([]byte(sn)))
	d := g.Record{
		"id":                newid,
		"prefix":            meta.Prefix,
		"oid":               order.OID,
		"uid":               order.UID,
		"username":          order.Username,
		"channel_id":        order.ChannelID,
		"cid":               order.CID,
		"pid":               order.PID,
		"amount":            amount,
		"state":             DepositConfirming,
		"automatic":         "0",
		"created_at":        fmt.Sprintf("%d", ca),
		"created_uid":       uid,
		"created_name":      name,
		"confirm_at":        "0",
		"confirm_uid":       "0",
		"confirm_name":      "",
		"review_remark":     remark,
		"finance_type":      helper.TransactionDeposit,
		"top_uid":           order.TopUID,
		"top_name":          order.TopName,
		"parent_uid":        order.ParentUID,
		"parent_name":       order.ParentName,
		"manual_remark":     order.ManualRemark,
		"bankcard_id":       order.BankcardID,
		"protocol_type":     order.ProtocolType,
		"rate":              order.Rate,
		"usdt_final_amount": order.USDTFinalAmount,
		"usdt_apply_amount": order.USDTApplyAmount,
		"address":           order.Address,
		"hash_id":           order.HashID,
		"flag":              order.Flag,
		"bank_code":         order.BankCode,
		"bank_no":           order.BankNo,
		"level":             order.Level,
		"tester":            tester,
		"r":                 mhash,
		"first_deposit_at":  mb.FirstDepositAt,
	}
	query, _, _ := dialect.Insert("tbl_deposit").Rows(d).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		fmt.Println("deposit err = ", err)
		_ = tx.Rollback()
		return errors.New(helper.TransErr)
	}

	// 判断状态，如果处理中则更新状态
	if order.State == DepositConfirming {
		// 更新配置
		ex := g.Ex{"id": order.ID, "prefix": meta.Prefix, "state": DepositConfirming}
		recs := g.Record{
			"state":         DepositCancelled,
			"confirm_at":    now.Unix(),
			"automatic":     "0",
			"confirm_uid":   uid,
			"confirm_name":  name,
			"review_remark": remark,
		}
		query, _, _ = dialect.Update("tbl_deposit").Set(recs).Where(ex).ToSQL()
		r, err := tx.Exec(query)
		fmt.Println(r)
		fmt.Println(err)
		if err != nil {
			_ = tx.Rollback()
			return errors.New(helper.TransErr)
		}
		refectRows, err := r.RowsAffected()
		if err != nil {
			_ = tx.Rollback()
			return errors.New(helper.TransErr)
		}
		fmt.Println(refectRows)

		if refectRows == 0 {
			_ = tx.Rollback()
			return errors.New(helper.TransErr)
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.New(helper.TransErr)
	}

	// 发送消息通知
	_ = PushMerchantNotify(manualReviewFmt, name, order.Username, amount)

	return nil
}

// DepositCallBack 存款回调
func DepositCallBack(fctx *fasthttp.RequestCtx) {

	var (
		err  error
		data paymentCallbackResp
	)

	pLog := paymentTDLog{
		Merchant:   "CGPAY",
		Flag:       "1",
		Lable:      paymentLogTag,
		RequestURL: string(fctx.RequestURI()),
	}

	if string(fctx.Method()) == fasthttp.MethodGet {
		pLog.RequestBody = fctx.QueryArgs().String()
	}

	if string(fctx.Method()) == fasthttp.MethodPost {
		pLog.RequestBody = fctx.PostArgs().String()
	}

	// 记录请求日志
	defer func() {
		if err != nil {
			pLog.Error = err.Error()
		}

		pLog.ResponseBody = string(fctx.Response.Body())
		pLog.ResponseCode = fctx.Response.StatusCode()
		paymentPushLog(pLog)
	}()

	// 获取并校验回调参数
	data, err = vnPay.PayCallBack(fctx)
	if err != nil {
		fctx.SetBody([]byte(`failed`))
		return
	}
	pLog.OrderID = data.OrderID

	// 查询订单
	order, err := depositFind(data.OrderID)
	if err != nil {
		err = fmt.Errorf("query order error: [%v]", err)
		fctx.SetBody([]byte(`failed`))
		return
	}

	pLog.Username = order.Username

	ch, err := ChannelTypeById(order.ChannelID)
	if err != nil {
		//return "", errors.New(helper.ChannelNotExist)

		return
	}

	pLog.Channel = ch["name"]

	if order.State == DepositSuccess || order.State == DepositCancelled {
		err = fmt.Errorf("duplicated deposite notify: [%d]", order.State)
		fctx.SetBody([]byte(`failed`))
		return
	}

	// 兼容越南盾的单位K 与 人民币元
	if data.Cent == 0 {
		data.Cent = 1000
	}

	orderAmount := fmt.Sprintf("%.4f", order.Amount)
	err = compareAmount(data.Amount, orderAmount, data.Cent)
	if err != nil {
		fmt.Printf("compare amount error: [err: %v, req: %s, origin: %s]", err, data.Amount, orderAmount)
		fctx.SetBody([]byte(`failed`))
		return
	}

	// 修改订单状态
	err = depositUpdate(data.State, order, data.PayAt)
	if err != nil {
		fmt.Printf("set order state error: [%v], old state=%d, new state=%d", err, order.State, data.State)
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

// 获取订单信息
func depositFind(id string) (Deposit, error) {

	d := Deposit{}

	ex := g.Ex{"id": id}
	query, _, _ := dialect.From("tbl_deposit").Select(colsDeposit...).Where(ex).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&d, query)
	if err == sql.ErrNoRows {
		return d, errors.New(helper.OrderNotExist)
	}

	if err != nil {
		return d, pushLog(err, helper.DBErr)
	}

	return d, nil
}

func depositUpdate(state int, order Deposit, payAt string) error {

	// 加锁
	err := depositLock(order.ID)
	if err != nil {
		return err
	}
	defer depositUnLock(order.ID)

	// 充值成功处理订单状态
	if state == DepositSuccess {
		_ = CacheDepositProcessingRem(order.UID)
		err = DepositUpPointSuccess(order.ID, "0", "", "", payAt, state)
		if err != nil {
			return err
		}
	} else {
		err = DepositUpPointCancel(order.ID, "0", "", "", payAt, state)
		if err != nil {
			return err
		}
	}

	return nil
}

// 存款上分
func DepositUpPointSuccess(did, uid, name, remark, payAt string, state int) error {

	// 判断状态是否合法
	allow := map[int]bool{
		DepositSuccess: true,
	}
	if _, ok := allow[state]; !ok {
		return errors.New(helper.OrderStateErr)
	}

	// 判断订单是否存在
	ex := g.Ex{"id": did, "state": DepositConfirming}
	order, err := DepositOrderFindOne(ex)
	if err != nil {
		return err
	}

	// 如果已经有一笔订单补单成功,则其他订单不允许补单成功
	if DepositSuccess == state {
		// 这里的ex不能覆盖上面的ex
		_, err = DepositOrderFindOne(g.Ex{"oid": order.OID, "state": DepositSuccess})
		if err != nil && err.Error() != helper.OrderNotExist {
			return err
		}

		if err == nil {
			return errors.New(helper.OrderExist)
		}
	}

	now := time.Now()
	if payAt != "" {
		confirmAt, err := strconv.ParseInt(payAt, 10, 64)
		if err == nil {
			if len(payAt) == 13 {
				confirmAt = confirmAt / 1000
			}
			now = time.Unix(confirmAt, 0)
		}
	}
	record := g.Record{
		"state":         state,
		"confirm_at":    now.Unix(),
		"confirm_uid":   uid,
		"confirm_name":  name,
		"review_remark": remark,
	}
	query, _, _ := dialect.Update("tbl_deposit").Set(record).Where(ex).ToSQL()
	fmt.Println(query)
	money := decimal.NewFromFloat(order.Amount)
	amount := money.String()
	cashType := helper.TransactionDeposit
	if money.Cmp(zero) == -1 {
		cashType = helper.TransactionFinanceDownPoint
		amount = money.Abs().String()
	}

	// 下分成功 修改订单状态并修改adjust表的审核状态
	if cashType == helper.TransactionFinanceDownPoint {
		//开启事务
		tx, err := meta.MerchantDB.Begin()
		if err != nil {
			return pushLog(err, helper.DBErr)
		}

		_, err = tx.Exec(query)
		if err != nil {
			_ = tx.Rollback()
			return pushLog(err, helper.DBErr)
		}

		r := g.Record{
			"state":          AdjustReviewPass,
			"review_at":      now.Unix(),
			"review_uid":     uid,
			"review_name":    name,
			"review_remark":  remark,
			"hand_out_state": AdjustSuccess,
		}
		query, _, _ = dialect.Update("tbl_member_adjust").Set(r).Where(g.Ex{"id": order.OID}).ToSQL()
		fmt.Println(query)
		_, err = tx.Exec(query)
		if err != nil {
			_ = tx.Rollback()
			return pushLog(err, helper.DBErr)
		}

		err = tx.Commit()
		if err != nil {
			return pushLog(err, helper.DBErr)
		}

		return nil
	}

	// 后面都是存款成功 和 下分失败 的处理
	// 1、查询用户额度
	balance, err := GetBalanceDB(order.UID)
	if err != nil {
		return err
	}
	balanceAfter := decimal.NewFromFloat(balance.Balance).Add(money)

	// 开启事务
	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	// 2、更新订单状态
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	balanceFeeAfter := balanceAfter
	fee := decimal.Zero
	var feeCashType int
	//如果存款有优惠
	key := meta.Prefix + ":f:p:" + order.PID
	promoDiscount, err := meta.MerchantRedis.HGet(ctx, key, "discount").Result()
	if err != nil && err != redis.Nil {
		//缓存没有配置就跳过
		fmt.Println(err)
	}
	pd, _ := decimal.NewFromString(promoDiscount)
	fmt.Println("promoDiscount:", promoDiscount)
	if pd.GreaterThan(decimal.Zero) {
		//大于0就是优惠，给钱
		fee = money.Mul(pd).Div(decimal.NewFromInt(100))
		money = money.Add(fee)
		balanceFeeAfter = decimal.NewFromFloat(balance.Balance).Add(money.Abs())
		feeCashType = helper.TransactionDepositBonus
	} else if pd.LessThan(decimal.Zero) {
		//小于0就是收费，扣钱
		fee = money.Mul(pd).Div(decimal.NewFromInt(100))
		money = money.Add(fee)
		balanceFeeAfter = decimal.NewFromFloat(balance.Balance).Add(money.Abs())
		feeCashType = helper.TransactionDepositFee

	}
	//修改存款订单的存款优惠
	record["discount"] = fee
	query, _, _ = dialect.Update("tbl_deposit").Set(record).Where(g.Ex{"id": order.ID}).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	// 3、更新余额
	ex = g.Ex{
		"uid":    order.UID,
		"prefix": meta.Prefix,
	}
	br := g.Record{
		"balance": g.L(fmt.Sprintf("balance+%s", money.String())),
	}
	query, _, _ = dialect.Update("tbl_members").Set(br).Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	// 4、新增账变记录
	id := helper.GenId()
	mbTrans := memberTransaction{
		AfterAmount:  balanceAfter.String(),
		Amount:       amount,
		BeforeAmount: decimal.NewFromFloat(balance.Balance).String(),
		BillNo:       order.ID,
		CreatedAt:    time.Now().UnixMilli(),
		ID:           id,
		CashType:     cashType,
		UID:          order.UID,
		Username:     order.Username,
		Prefix:       meta.Prefix,
	}
	if cashType == helper.TransactionFinanceDownPoint {
		mbTrans.OperationNo = id
	}

	query, _, _ = dialect.Insert("tbl_balance_transaction").Rows(mbTrans).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	if balanceFeeAfter.Cmp(balanceAfter) != 0 {
		//手续费/优惠的帐变
		id = helper.GenId()
		mbTrans = memberTransaction{
			AfterAmount:  balanceFeeAfter.String(),
			Amount:       fee.String(),
			BeforeAmount: balanceAfter.String(),
			BillNo:       order.ID,
			CreatedAt:    time.Now().UnixMilli(),
			ID:           id,
			CashType:     feeCashType,
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
	}

	err = tx.Commit()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	rec := g.Record{
		"first_deposit_at":     now.Unix(),
		"first_deposit_amount": order.Amount,
	}
	ex1 := g.Ex{
		"username":         order.Username,
		"first_deposit_at": 0,
	}
	query, _, _ = dialect.Update("tbl_members").Set(rec).Where(ex1).ToSQL()
	fmt.Printf("memberFirstDeposit Update: %v\n", query)
	result, err := meta.MerchantDB.Exec(query)
	if err != nil {
		fmt.Println("update member first_amount err:", err.Error())
	}
	updateRows, _ := result.RowsAffected()
	if updateRows == 0 {
		rec = g.Record{
			"second_deposit_at":     now.Unix(),
			"second_deposit_amount": order.Amount,
		}
		ex = g.Ex{
			"username":          order.Username,
			"second_deposit_at": 0,
		}
		query, _, _ = dialect.Update("tbl_members").Set(rec).Where(ex).ToSQL()
		fmt.Printf("memberSecondDeposit Update: %v\n", query)
		_, err = meta.MerchantDB.Exec(query)
		if err != nil {
			fmt.Println("update member second_amount err:", err.Error())
		}
	}

	_ = RocketSendAsync(meta.Prefix+"_finish_flow", []byte("D_"+order.ID))
	//发送推送
	msg := fmt.Sprintf(`{"ty":"1","amount": "%f", "ts":"%d","status":"success"}`, order.Amount, now.Unix())
	fmt.Println(msg)
	topic := fmt.Sprintf("%s/%s/finance", meta.Prefix, order.UID)
	err = Publish(topic, []byte(msg))
	if err != nil {
		fmt.Println("merchantNats.Publish finance = ", err.Error())
		return err
	}

	lastDepositKey := fmt.Sprintf("%s:uld:%s", meta.Prefix, order.Username)
	_ = meta.MerchantRedis.Set(ctx, lastDepositKey, order.ChannelID, 100*time.Hour).Err()
	lastDepositPaymentKey := fmt.Sprintf("%s:uldp:%s", meta.Prefix, order.Username)
	_ = meta.MerchantRedis.Set(ctx, lastDepositPaymentKey, order.PID, 100*time.Hour).Err()

	_ = MemberUpdateCache(order.Username)

	return nil
}

// 存款上分
func DepositUpPointCancel(did, uid, name, remark, payAt string, state int) error {

	// 判断状态是否合法
	allow := map[int]bool{
		DepositCancelled: true,
	}
	if _, ok := allow[state]; !ok {
		return errors.New(helper.OrderStateErr)
	}

	// 判断订单是否存在
	ex := g.Ex{"id": did, "state": DepositConfirming}
	order, err := DepositOrderFindOne(ex)
	if err != nil {
		return err
	}

	// 如果已经有一笔订单补单成功,则其他订单不允许补单成功
	if DepositSuccess == state {
		// 这里的ex不能覆盖上面的ex
		_, err = DepositOrderFindOne(g.Ex{"oid": order.OID, "state": DepositSuccess})
		if err != nil && err.Error() != helper.OrderNotExist {
			return err
		}

		if err == nil {
			return errors.New(helper.OrderExist)
		}
	}

	now := time.Now()
	if payAt != "" {
		confirmAt, err := strconv.ParseInt(payAt, 10, 64)
		if err == nil {
			if len(payAt) == 13 {
				confirmAt = confirmAt / 1000
			}
			now = time.Unix(confirmAt, 0)
		}
	}
	record := g.Record{
		"state":         state,
		"confirm_at":    now.Unix(),
		"confirm_uid":   uid,
		"confirm_name":  name,
		"review_remark": remark,
	}
	query, _, _ := dialect.Update("tbl_deposit").Set(record).Where(ex).ToSQL()
	fmt.Println(query)
	money := decimal.NewFromFloat(order.Amount)
	amount := money.String()
	cashType := helper.TransactionDeposit
	if money.Cmp(zero) == -1 {
		cashType = helper.TransactionFinanceDownPoint
		amount = money.Abs().String()
	}

	// 存款失败 直接修改订单状态
	if cashType == helper.TransactionDeposit {
		_, err = meta.MerchantDB.Exec(query)
		if err != nil {
			return pushLog(err, helper.DBErr)
		}
		//发送推送
		msg := fmt.Sprintf(`{"ty":"1","amount": "%f", "ts":"%d","status":"failed"}`, order.Amount, time.Now().Unix())
		fmt.Println(msg)
		topic := fmt.Sprintf("%s/%s/finance", meta.Prefix, order.UID)
		err = Publish(topic, []byte(msg))
		if err != nil {
			fmt.Println("merchantNats.Publish finance = ", err.Error())
			return err
		}
		return nil
	} else if cashType == helper.TransactionFinanceDownPoint {
		money = money.Abs()
	}

	// 后面都是存款成功 和 下分失败 的处理
	// 1、查询用户额度
	balance, err := GetBalanceDB(order.UID)
	if err != nil {
		return err
	}
	balanceAfter := decimal.NewFromFloat(balance.Balance).Add(money)

	// 开启事务
	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	// 2、更新订单状态
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	// 如果是下分 审核失败
	if DepositCancelled == state && cashType == helper.TransactionFinanceDownPoint {
		// 修改状态
		r := g.Record{
			"state":          AdjustReviewReject,
			"review_at":      now.Unix(),
			"review_uid":     uid,
			"review_name":    name,
			"review_remark":  remark,
			"hand_out_state": AdjustFailed,
		}
		query, _, _ = dialect.Update("tbl_member_adjust").Set(r).Where(g.Ex{"id": order.OID}).ToSQL()
		_, err = tx.Exec(query)
		if err != nil {
			_ = tx.Rollback()
			return pushLog(err, helper.DBErr)
		}

		balanceAfter = decimal.NewFromFloat(balance.Balance).Add(money.Abs())
	}

	balanceFeeAfter := balanceAfter
	fee := decimal.Zero
	var feeCashType int
	//如果存款有优惠
	key := meta.Prefix + ":f:p:" + order.PID

	promoDiscount, err := meta.MerchantRedis.HGet(ctx, key, "discount").Result()
	if err != nil && err != redis.Nil {
		//缓存没有配置就跳过
		fmt.Println(err)
	}
	pd, _ := decimal.NewFromString(promoDiscount)
	fmt.Println("promoDiscount:", promoDiscount)
	if pd.GreaterThan(decimal.Zero) {
		//大于0就是优惠，给钱
		fee = money.Mul(pd).Div(decimal.NewFromInt(100))
		money = money.Add(fee)
		balanceFeeAfter = decimal.NewFromFloat(balance.Balance).Add(money.Abs())
		feeCashType = helper.TransactionDepositBonus
	} else if pd.LessThan(decimal.Zero) {
		//小于0就是收费，扣钱
		fee = money.Mul(pd).Div(decimal.NewFromInt(100))
		money = money.Add(fee)
		balanceFeeAfter = decimal.NewFromFloat(balance.Balance).Add(money.Abs())
		feeCashType = helper.TransactionDepositFee

	}
	//修改存款订单的存款优惠
	record["discount"] = fee
	query, _, _ = dialect.Update("tbl_deposit").Set(record).Where(g.Ex{"id": order.ID}).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	// 3、更新余额
	ex = g.Ex{
		"uid":    order.UID,
		"prefix": meta.Prefix,
	}
	br := g.Record{
		"balance": g.L(fmt.Sprintf("balance+%s", money.String())),
	}
	query, _, _ = dialect.Update("tbl_members").Set(br).Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	// 4、新增账变记录
	id := helper.GenId()
	mbTrans := memberTransaction{
		AfterAmount:  balanceAfter.String(),
		Amount:       amount,
		BeforeAmount: decimal.NewFromFloat(balance.Balance).String(),
		BillNo:       order.ID,
		CreatedAt:    time.Now().UnixMilli(),
		ID:           id,
		CashType:     cashType,
		UID:          order.UID,
		Username:     order.Username,
		Prefix:       meta.Prefix,
	}
	if cashType == helper.TransactionFinanceDownPoint {
		mbTrans.OperationNo = id
	}

	query, _, _ = dialect.Insert("tbl_balance_transaction").Rows(mbTrans).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	if balanceFeeAfter.Cmp(balanceAfter) != 0 {
		//手续费/优惠的帐变
		id = helper.GenId()
		mbTrans = memberTransaction{
			AfterAmount:  balanceFeeAfter.String(),
			Amount:       fee.String(),
			BeforeAmount: balanceAfter.String(),
			BillNo:       order.ID,
			CreatedAt:    time.Now().UnixMilli(),
			ID:           id,
			CashType:     feeCashType,
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
	}

	err = tx.Commit()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	//发送推送
	msg := fmt.Sprintf(`{"ty":"1","amount": "%f", "ts":"%d","status":"failed"}`, order.Amount, now.Unix())
	fmt.Println(msg)
	topic := fmt.Sprintf("%s/%s/finance", meta.Prefix, order.UID)
	err = Publish(topic, []byte(msg))
	if err != nil {
		fmt.Println("merchantNats.Publish finance = ", err.Error())
		return err
	}

	_ = MemberUpdateCache(order.Username)

	return nil
}
