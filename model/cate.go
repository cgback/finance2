package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
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
		"id": id,
	}
	query, _, _ := dialect.From("f2_category").Select(colCate...).Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&cate, query)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println(err)
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

func CateListByName(name string) (Category, error) {

	var data Category

	cond := g.Ex{"name": name}
	query, _, _ := dialect.From("f2_category").Select(colCate...).Where(cond).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&data, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	return data, nil
}

func cateMap() (map[string]Category, error) {

	m := map[string]Category{}
	var data []Category

	cond := g.Ex{}
	query, _, _ := dialect.From("f2_category").Select(colCate...).Where(cond).ToSQL()
	err := meta.MerchantDB.Select(&data, query)
	if err != nil {
		return m, pushLog(err, helper.DBErr)
	}
	for _, v := range data {
		m[v.ID] = v
	}

	return m, nil
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

	CacheRefreshLevel()

	cateToRedis()
	return nil
}

func CateWithdrawList(amount float64) ([]Category, error) {

	var data []Category

	ex := g.Ex{
		"id":    []string{"59000000000000101"},
		"state": "1",
	}
	if amount != 0 {
		ex["fmin"] = g.Op{"lte": amount}
		ex["fmax"] = g.Op{"gte": amount}
	}

	var pids []string
	query, _, _ := dialect.From("f2_payment").Select("cate_id").Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&pids, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	if len(pids) == 0 {
		return data, nil
	}

	ex = g.Ex{
		"id":    pids,
		"state": "1",
	}
	query, _, _ = dialect.From("f2_category").Select(colCate...).Where(ex).Order(g.C("created_at").Desc()).ToSQL()
	err = meta.MerchantDB.Select(&data, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	return data, nil
}
