package model

import (
	"errors"
	"finance/contrib/helper"
	"finance/contrib/validator"
	"fmt"
	"github.com/go-redis/redis/v8"
	"strconv"
	"time"

	g "github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/shopspring/decimal"
	"github.com/tenfyzhong/cityhash"
	"github.com/valyala/fasthttp"
)

// Manual 调用与pid对应的渠道, 发起充值(代付)请求
func OfflinePay(fctx *fasthttp.RequestCtx, paymentID, amount, bid string) (string, error) {

	res := map[string]string{}
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
					return "", errors.New(fmt.Sprintf("Thao Tác quá nhanh, vui lòng thử lại sau %d giây!", int64(dttb)-tts))
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
					return "", errors.New(fmt.Sprintf("Thao Tác quá nhanh, vui lòng thử lại sau %d giây!", int64(dtta)-tts))
				}
			}
			dtom := cd["deposit_time_one_max"]
			dto := cd["deposit_time_one"]
			dtomi, _ := strconv.ParseInt(dtom, 10, 64)
			dtoi, _ := strconv.Atoi(dto)
			if total.T.Int64 >= dtomi {
				tts := time.Now().Unix() - total.L.Int64
				if tts < int64(dtoi) {
					return "", errors.New(fmt.Sprintf("Thao Tác quá nhanh, vui lòng thử lại sau %d giây!", int64(dtoi)-tts))
				}
			}
		}
	}

	ts := fctx.Time().In(loc).Unix()
	p, err := CachePayment(paymentID)
	if err != nil {
		return "", errors.New(helper.ChannelNotExist)
	}

	// 检查存款金额是否符合范围
	a, ok := validator.CheckFloatScope(amount, p.Fmin, p.Fmax)
	if !ok {
		return "", errors.New(helper.AmountOutRange)
	}

	amount = a.Truncate(0).String()
	var bc Bankcard_t
	if bid != "" {
		bc, err = BankCardBackendById(bid)
		if err != nil {
			fmt.Println("BankCardBackend err = ", err.Error())
			return "", errors.New(helper.BankCardNotExist)
		}
	} else {
		bc, err = BankCardBackend(p.ID)
		if err != nil {
			fmt.Println("BankCardBackend err = ", err.Error())
			return "", errors.New(helper.BankCardNotExist)
		}
	}

	// 获取附言码
	code, err := transacCodeGet()
	if err != nil {
		return "", errors.New(helper.ChannelBusyTryOthers)
	}

	fmt.Println("TransacCodeGet code = ", code)
	skey := meta.Prefix + ":f:p:" + p.ID
	promoDiscount, err := meta.MerchantRedis.HGet(ctx, skey, "discount").Result()
	if err != nil && err != redis.Nil {
		//缓存没有配置就跳过
		fmt.Println(err)
	}
	fmt.Println("discount:", promoDiscount)
	pd, _ := decimal.NewFromString(promoDiscount)
	am, _ := decimal.NewFromString(amount)
	discount := pd.Mul(am).Div(decimal.NewFromInt(100))
	// 生成我方存款订单号
	orderId := helper.GenId()
	now := time.Now()
	// 生成订单
	ca := now.Unix()
	sn := fmt.Sprintf(`deposit%s%s%d%d`, orderId, user.Username, ca, user.CreatedAt)
	mhash := fmt.Sprintf("%d", cityhash.CityHash64([]byte(sn)))
	d := g.Record{
		"id":            orderId,
		"prefix":        meta.Prefix,
		"oid":           orderId,
		"uid":           user.UID,
		"top_uid":       user.TopUid,
		"top_name":      user.TopName,
		"parent_name":   user.ParentName,
		"parent_uid":    user.ParentUid,
		"username":      user.Username,
		"channel_id":    p.ChannelID,
		"cid":           p.CateID,
		"pid":           p.ID,
		"amount":        amount,
		"state":         DepositConfirming,
		"finance_type":  helper.TransactionOfflineDeposit,
		"automatic":     "0",
		"created_at":    ts,
		"created_uid":   "0",
		"created_name":  "",
		"confirm_at":    "0",
		"confirm_uid":   "0",
		"confirm_name":  "",
		"review_remark": "",
		"manual_remark": fmt.Sprintf(`{"manual_remark": "%s", "real_name":"%s", "bank_addr":"%s", "name":"%s"}`, code, bc.AccountName, bc.BankcardAddr, bc.BanklcardName),
		"bankcard_id":   bc.Id,
		"flag":          "3",
		//"bank_code":     bankCode,
		"bank_no":          bc.BanklcardNo,
		"level":            user.Level,
		"tester":           user.Tester,
		"r":                mhash,
		"discount":         discount.StringFixed(2),
		"first_deposit_at": user.FirstDepositAt,
	}

	// 请求成功插入订单
	err = deposit(d)
	if err != nil {
		fmt.Println("Manual deposit err = ", err)
		return "", pushLog(err, helper.DBErr)
	}

	res = map[string]string{
		"id":           orderId,
		"name":         bc.BanklcardName,
		"cardNo":       bc.BanklcardNo,
		"realname":     bc.AccountName,
		"manualRemark": code,
		"ts":           fmt.Sprintf("%d", ts),
		"bid":          bc.Id,
		"useLink":      "0",
		"is_qr":        bc.Flags, //1有二维码 2 没有二维码
	}

	bytes, _ := helper.JsonMarshal(res)

	// 发送消息通知
	_ = PushWithdrawNotify(depositReviewFmt, user.Username, amount)

	if user.Tester == "0" && user.TopName != "p32015" && user.TopName != "nanfeng001" {
		msg := map[string]string{}
		msg["order_id"] = orderId
		_ = rocketPutDelay("recharge_tester", msg, randSliceValue([]int{14, 15, 16, 17, 18}))
	}
	return string(bytes), nil
}

// DepositManualList 线下转卡订单列表
func ManualList(ex g.Ex, startTime, endTime string, isBig, firstWd, page, pageSize int) (mFDepositData, error) {

	ex["prefix"] = meta.Prefix
	ex["tester"] = 1
	data := mFDepositData{}

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
	orderTemp := "created_at"
	cols := colsDeposit
	cd, err := ConfigDetail()
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}
	depositListFirst := cd["deposit_list_first"]
	if isBig == 1 && firstWd == 1 {
		cols = append(cols, g.L("case when first_deposit_at = 0 then 2000000000+created_at  when amount > "+depositListFirst+" then (created_at+1000000000) else created_at end as sort_num"))
		orderTemp = "sort_num"
	}
	if isBig == 1 {
		cols = append(cols, g.L("case when amount > "+depositListFirst+" then (created_at+1000000000) else created_at end as sort_num"))
		orderTemp = "sort_num"
	}
	if firstWd == 1 {
		cols = append(cols, g.L("case when first_deposit_at = 0 then 2000000000+created_at else created_at end as sort_num"))
		orderTemp = "sort_num"
	}
	offset := uint((page - 1) * pageSize)
	query, _, _ := dialect.From("tbl_deposit").Select(cols...).
		Where(ex).Offset(offset).Limit(uint(pageSize)).Order(g.C(orderTemp).Desc()).ToSQL()
	fmt.Println(query)
	err = meta.MerchantDB.Select(&data.D, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	return data, nil
}

// DepositManualReview 线下转卡-存款审核
func ManualReview(did, remark, name, uid string, state int, record Deposit) error {

	// 加锁
	err := depositLock(did)
	if err != nil {
		return err
	}
	defer depositUnLock(did)

	if state == DepositSuccess {
		err = DepositUpPointReviewSuccess(did, uid, name, remark, state)

		// 清除未未成功的订单计数
		amount := decimal.NewFromFloat(record.Amount)

		vals := g.Record{
			"daily_finish_amount": g.L(fmt.Sprintf("daily_finish_amount+%s", amount.String())),
		}
		err = BankCardUpdate(record.BankcardID, vals)
		if err != nil {
			fmt.Println("ManualReview BankCardUpdate = ", err)
			return err
		}

		bc, err := BankCardByID(record.BankcardID)
		if err == nil {
			dailyFinishAmount, _ := decimal.NewFromString(bc.DailyFinishAmount)
			dailyMaxAmount, _ := decimal.NewFromString(bc.DailyMaxAmount)

			if dailyFinishAmount.Cmp(dailyMaxAmount) >= 0 {

				vals = g.Record{
					"state": "0",
				}
				BankCardUpdate(record.BankcardID, vals)
			}
		}
	} else if state == DepositCancelled {
		err = DepositUpPointReviewCancel(did, uid, name, remark, state)
	}
	if err == nil {

		key := fmt.Sprintf("%s:finance:manual:c:%s", meta.Prefix, record.Username)
		err = meta.MerchantRedis.ZRem(ctx, key, did).Err()
		if err != nil {
			_ = pushLog(err, helper.RedisErr)
		}
		return nil
	}

	return err
}

func GenCode(fctx *fasthttp.RequestCtx, amount, bid, code string) (string, error) {

	res := map[string]string{}
	_, err := MemberCache(fctx)
	if err != nil {
		return "", err
	}

	var bc Bankcard_t
	if bid != "" {
		bc, err = BankCardBackendById(bid)
		if err != nil {
			fmt.Println("BankCardBackend err = ", err.Error())
			return "", errors.New(helper.BankCardNotExist)
		}
	} else {
		return "", errors.New(helper.BankCardNotExist)
	}

	qr, err := QrCodeGen(bc, amount, code)

	res = map[string]string{
		"qr": qr,
	}

	bytes, _ := helper.JsonMarshal(res)

	return string(bytes), nil
}
