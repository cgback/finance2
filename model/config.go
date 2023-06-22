package model

import (
	"errors"
	"finance/contrib/helper"
	g "github.com/doug-martin/goqu/v9"
)

func ConfigDetail() (map[string]string, error) {

	res := map[string]string{}
	var data []FConfig
	query, _, _ := dialect.From("f_config").Select(colsConfig...).ToSQL()
	err := meta.MerchantDB.Select(&data, query)
	if err != nil {
		return res, pushLog(err, helper.DBErr)
	}

	for _, val := range data {
		res[val.Name] = val.Content
	}

	return res, nil
}

func ConfigUpdate(configs map[string]string) error {

	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return errors.New(helper.TransErr)
	}
	for key, value := range configs {
		record := g.Record{
			"content": value,
		}
		query, _, _ := dialect.Update("f_config").Where(g.Ex{"name": key}).Set(record).ToSQL()
		_, err = tx.Exec(query)
		if err != nil {
			tx.Rollback()
			return pushLog(err, helper.DBErr)
		}
	}
	tx.Commit()
	return nil
}
