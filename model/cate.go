package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	g "github.com/doug-martin/goqu/v9"
	"strings"
)

type Category struct {
	ID         string `db:"id" json:"id"`
	Name       string `db:"name" json:"name"`
	MerchantId string `db:"merchant_id" json:"merchant_id"`
	State      string `db:"state" json:"state"`
	Comment    string `db:"comment" json:"comment"`
	CreatedAt  int64  `db:"created_at" json:"created_at"`
	Prefix     string `db:"prefix" json:"prefix"`
}

// CateIDAndName 渠道id和name
type CateIDAndName struct {
	ID   string `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
}

// 三方渠道
func CateByID(id string) (Category, error) {

	var cate Category
	ex := g.Ex{
		"prefix": meta.Prefix,
		"id":     id,
	}
	query, _, _ := dialect.From("f2_category").Select(colCate...).Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&cate, query)
	if err != nil && err != sql.ErrNoRows {
		return cate, pushLog(err, helper.DBErr)
	}

	return cate, nil
}

// CateIDAndNameByCIDS 通过cid查询渠道id和渠道名
func CateIDAndNameByCIDS(cids []string) (map[string]CateIDAndName, error) {

	var (
		data []CateIDAndName
		res  = make(map[string]CateIDAndName)
	)

	if len(cids) == 0 {
		return res, nil
	}

	ex := g.Ex{
		"id":     g.Op{"in": cids},
		"prefix": meta.Prefix,
	}
	query, _, _ := dialect.From("f2_category").Select("id", "name").Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&data, query)
	if err != nil {
		return res, pushLog(err, helper.DBErr)
	}

	for _, v := range data {
		if _, ok := res[v.ID]; !ok {
			res[v.ID] = v
		}
	}

	return res, nil
}

func CateList() ([]Category, error) {

	var data []Category

	cond := g.Ex{}
	query, _, _ := dialect.From("f2_category").Select(colCate...).Where(cond).ToSQL()
	err := meta.MerchantDB.Select(&data, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	return data, nil
}

// 设置渠道的状态 开启/关闭
func CateSet(id, state string) error {

	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return errors.New(helper.TransErr)
	}

	ex := g.Ex{
		"id": id,
	}

	query, _, _ := dialect.Update("f2_category").Set(g.Record{"state": state}).Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return errors.New(helper.TransErr)
	}

	// 切换到关闭状态，旗下所有支付方式也将同时切换到关闭状态
	if state == "0" {
		ex = g.Ex{
			"prefix":  meta.Prefix,
			"cate_id": id,
		}
		query, _, _ = dialect.Update("f2_payment").Set(g.Record{"state": state}).Where(ex).ToSQL()
		_, err = tx.Exec(query)
		if err != nil {
			_ = tx.Rollback()
			return errors.New(helper.TransErr)
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.New(helper.TransErr)
	}

	var pl []Payment_t
	lm := map[string]struct{}{}
	ex = g.Ex{
		"cate_id": id,
	}
	query, _, _ = dialect.From("f2_payment").Select(colsPayment...).Where(ex).ToSQL()
	err = meta.MerchantDB.Select(&pl, query)
	if err == nil {
		for _, level := range pl {
			ls := strings.Split(level.VipList, ",")
			for _, l := range ls {
				lm[l] = struct{}{}
			}
		}
	}

	CacheRefreshLevel()

	cateToRedis()
	return nil
}
