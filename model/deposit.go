package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
	"github.com/go-redis/redis/v8"
	"github.com/shopspring/decimal"
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
	key := meta.Prefix + ":p:c:t:" + order.ChannelID
	promoState, err := meta.MerchantRedis.HGet(ctx, key, "promo_state").Result()
	if err != nil && err != redis.Nil {
		//缓存没有配置就跳过
		fmt.Println(err)
	}
	//开启了优惠
	if promoState == "1" {
		promoDiscount, err := meta.MerchantRedis.HGet(ctx, key, "promo_discount").Result()
		if err != nil && err != redis.Nil {
			//缓存没有配置就跳过
			fmt.Println(err)
		}
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
	if promoState == "1" {
		record["discount"] = fee
	}

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
