package model

import (
	"database/sql"
	"finance/contrib/helper"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
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
