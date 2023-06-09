package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
	"github.com/go-redis/redis/v8"
	"time"
)

func ChannelTypeList() ([]ChannelType, error) {

	var data []ChannelType
	query, _, _ := dialect.From("f2_channel_type").Select(colsChannelType...).Where(g.Ex{}).ToSQL()
	fmt.Println(query)
	err := meta.MerchantDB.Select(&data, query)
	if err != nil {
		return data, pushLog(fmt.Errorf("%s,[%s]", err.Error(), query), helper.DBErr)
	}

	return data, nil
}

func ChannelTypeUpdateState(id, state string, admin map[string]string) error {

	ex := g.Ex{
		"id": id,
	}
	data := ChannelType{}
	query, _, _ := dialect.From("f2_channel_type").Select(colsChannelType...).Where(ex).ToSQL()
	fmt.Println(query)
	err := meta.MerchantDB.Get(&data, query)
	if err != nil && err != sql.ErrNoRows {
		return pushLog(fmt.Errorf("%s,[%s]", err.Error(), query), helper.DBErr)
	}

	if err == sql.ErrNoRows {
		return errors.New(helper.RecordNotExistErr)
	}

	if state == data.State {
		return errors.New(helper.NoDataUpdate)
	}

	record := g.Record{
		"state":        state,
		"updated_at":   time.Now().Unix(),
		"updated_name": admin["name"],
		"updated_uid":  admin["id"],
	}
	query, _, _ = dialect.Update("f2_channel_type").Set(record).Where(g.Ex{"id": id}).ToSQL()
	fmt.Println(query)
	_, err = meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(fmt.Errorf("%s,[%s]", err.Error(), query), helper.DBErr)
	}

	LoadChannelType()
	if state == "2" {
		list, err := ChanByChan(id)
		if err == nil {
			for _, v := range list {
				ChannelSet(v.ID, "0", admin["id"], admin["name"])
			}
		}
	}
	return nil
}

func ChannelTypeUpdateSort(id string, sort int) error {

	ex := g.Ex{
		"id": id,
	}
	data := ChannelType{}
	query, _, _ := dialect.From("f2_channel_type").Select(colsChannelType...).Where(ex).ToSQL()
	fmt.Println(query)
	err := meta.MerchantDB.Get(&data, query)
	if err != nil && err != sql.ErrNoRows {
		return pushLog(fmt.Errorf("%s,[%s]", err.Error(), query), helper.DBErr)
	}

	if err == sql.ErrNoRows {
		return errors.New(helper.RecordNotExistErr)
	}

	if sort == data.Sort {
		return errors.New(helper.NoDataUpdate)
	}

	record := g.Record{
		"sort": sort,
	}
	query, _, _ = dialect.Update("f2_channel_type").Set(record).Where(g.Ex{"id": id}).ToSQL()
	fmt.Println(query)
	_, err = meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(fmt.Errorf("%s,[%s]", err.Error(), query), helper.DBErr)
	}

	LoadChannelType()

	return nil
}

func LoadChannelType() {

	cts, err := ChannelTypeList()
	if err == nil {
		pipe := meta.MerchantRedis.Pipeline()
		defer pipe.Close()

		var zs []*redis.Z
		key := fmt.Sprintf("%s:f2:channeltypes", meta.Prefix)
		pipe.Del(ctx, key)
		for _, v := range cts {
			fmt.Println(v.Name, ":", v.Sort)
			if v.State == "1" {
				z := &redis.Z{
					Score:  float64(v.Sort),
					Member: v.ID,
				}
				zs = append(zs, z)
			}
			cid := meta.Prefix + ":p:c:t:" + v.ID
			meta.MerchantRedis.HSet(ctx, cid, "name", v.Name, "id", v.ID, "sort", v.Sort)
		}
		if len(zs) > 0 {
			pipe.ZAdd(ctx, key, zs...)
		}
		_, _ = pipe.Exec(ctx)
	}
}

// 获取三方通道
func TunnelByID(id string) (ChannelType, error) {

	tunnel := ChannelType{}
	query, _, _ := dialect.From("f2_channel_type").Select(colsChannelType...).Where(g.Ex{"id": id}).ToSQL()
	err := meta.MerchantDB.Get(&tunnel, query)
	if err != nil && err != sql.ErrNoRows {
		return tunnel, pushLog(err, helper.DBErr)
	}

	return tunnel, nil
}

func ChannelTypeById(id string) (map[string]string, error) {

	key := meta.Prefix + ":p:c:t:" + id
	field := []string{"sort", "name", "id"}
	res := map[string]string{}

	pipe := meta.MerchantRedis.Pipeline()
	ex_temp := pipe.Exists(ctx, key)
	rs_temp := pipe.HMGet(ctx, key, field...)
	_, err := pipe.Exec(ctx)
	pipe.Close()

	if err != nil {
		return res, errors.New(helper.RedisErr)
	}

	if ex_temp.Val() == 0 {
		return res, errors.New(helper.RecordNotExistErr)
	}

	recs := rs_temp.Val()

	for k, v := range field {

		if val, ok := recs[k].(string); ok {
			res[v] = val
		} else {
			res[v] = ""
		}
	}

	return res, nil
}
