package model

import (
	"database/sql"
	"finance/contrib/helper"
	g "github.com/doug-martin/goqu/v9"
	"github.com/go-redis/redis/v8"
	"strings"
)

type PaymentIDChannelID struct {
	PaymentID string `db:"id" json:"id"`
	ChannelId string `db:"channel_id" json:"channel_id"`
}

type ChannelDevice struct {
	ID        string `db:"id" json:"id"`
	PaymentId string `db:"payment_id" json:"payment_id"`
	DeviceId  string `db:"device_id" json:"device_id"`
}

type channelCate struct {
	PaymentID string `db:"id" json:"id"`
	CateID    string `db:"cate_id" json:"cate_id"`
}

func PaymentList(cateID, chanID string) ([]Payment_t, error) {

	var data []Payment_t

	ex := g.Ex{
		"prefix": meta.Prefix,
	}

	if cateID != "0" {
		ex["cate_id"] = cateID
	}

	if chanID != "0" {
		ex["channel_id"] = chanID
	}

	query, _, _ := dialect.From("f_payment").Select(colPayment...).Where(ex).Order(g.C("cate_id").Desc()).ToSQL()

	err := meta.MerchantDB.Select(&data, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}
	ll := len(data)

	if ll > 0 {

		res := make([]*redis.StringCmd, ll)
		pipe := meta.MerchantRedis.Pipeline()
		for i, v := range data {
			key := meta.Prefix + ":p:c:t:" + v.ChannelID
			res[i] = pipe.HGet(ctx, key, "name")
		}

		pipe.Exec(ctx)
		pipe.Close()

		for i := 0; i < ll; i++ {
			data[i].ChannelName = res[i].Val()
		}
	}

	return data, nil
}

func ChannelUpdate(param map[string]string) error {

	record := g.Record{
		"fmin":         param["fmin"],
		"fmax":         param["fmax"],
		"payment_name": param["payment_name"],
		"st":           param["st"],
		"et":           param["et"],
		"sort":         param["sort"],
		"comment":      param["comment"],
		"amount_list":  param["amount_list"],
	}

	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return pushLog(err, helper.TransErr)
	}

	ex := g.Ex{
		"prefix": meta.Prefix,
		"id":     param["id"],
	}

	query, _, _ := dialect.Update("f_payment").Set(record).Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.TransErr)
	}

	err = tx.Commit()
	if err != nil {
		return pushLog(err, helper.TransErr)
	}

	_ = CacheRefreshPayment(param["id"])

	return nil
}

func ChannelSet(id, state string) error {

	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return pushLog(err, helper.TransErr)
	}

	ex := g.Ex{
		"prefix": meta.Prefix,
		"id":     id,
	}
	query, _, _ := dialect.Update("f2_payment").Set(g.Record{"state": state}).Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.TransErr)
	}

	err = tx.Commit()
	if err != nil {
		return pushLog(err, helper.TransErr)
	}

	p, err := ChanByID(id)
	vips := strings.Split(p.VipList, ",")
	for _, level := range vips {
		Create(level)
	}

	_ = CacheRefreshPayment(id)

	_ = CacheRefreshPaymentBanks(id)

	return nil
}

// ChanByCateAndChan 通过cate id和channel id查找cate
func ChanByCateAndChan(cateId, ChanId string) (Payment_t, error) {

	var channel Payment_t

	query, _, _ := dialect.From("f2_payment").Select(colPayment...).
		Where(g.Ex{"cate_id": cateId, "channel_id": ChanId, "prefix": meta.Prefix}).ToSQL()
	err := meta.MerchantDB.Get(&channel, query)
	if err != nil && err != sql.ErrNoRows {
		return channel, pushLog(err, helper.DBErr)
	}

	return channel, nil
}

func ChanByID(id string) (Payment_t, error) {

	var channel Payment_t

	ex := g.Ex{
		"id":     id,
		"prefix": meta.Prefix,
	}

	query, _, _ := dialect.From("f2_payment").Select(colPayment...).Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&channel, query)
	if err != nil && err != sql.ErrNoRows {
		return channel, pushLog(err, helper.DBErr)
	}

	return channel, nil
}

func ChanExistsByID(id string) (Payment_t, error) {

	var channel Payment_t
	ex := g.Ex{
		"prefix": meta.Prefix,
		"id":     id,
	}
	query, _, _ := dialect.From("f2_payment").Select(colPayment...).Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&channel, query)
	if err != nil && err != sql.ErrNoRows {
		return channel, pushLog(err, helper.DBErr)
	}

	return channel, nil
}
