package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	g "github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/shopspring/decimal"
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
