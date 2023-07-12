package model

import (
	"errors"
	"finance/contrib/helper"
	"finance/contrib/validator"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
	"strings"
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

func MemberConfigList(flag, usernames string) ([]FMemberConfig, error) {

	var data []FMemberConfig
	ex := g.Ex{"flag": flag}
	if usernames != "" {
		unames := strings.Split(usernames, ",")
		ex["username"] = unames
	}
	query, _, _ := dialect.From("f_member_config").Select(colsMemberConfig...).Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&data, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}
	return data, nil
}

func MemberConfigInsert(flag, usernames string) error {

	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return errors.New(helper.TransErr)
	}
	if usernames != "" {
		unames := strings.Split(usernames, ",")
		for _, username := range unames {
			if !validator.CheckUName(username, 5, 14) {
				return errors.New(helper.UsernameErr)

			}

			mb, err := MemberFindOne(username)
			if err != nil {
				return errors.New(helper.UserNotExist)
			}
			record := g.Record{
				"id":       helper.GenId(),
				"username": mb.Username,
				"uid":      mb.UID,
				"flag":     flag,
			}
			query, _, _ := dialect.Insert("f_member_config").Rows(record).ToSQL()
			fmt.Println(query)
			_, err = tx.Exec(query)
			if err != nil {
				_ = tx.Rollback()
				return pushLog(err, helper.DBErr)
			}
		}
	}

	tx.Commit()
	return nil
}

func MemberConfigDelete(id string) error {

	ex := g.Ex{
		"id": id,
	}
	query, _, _ := dialect.Delete("f_member_config").Where(ex).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}
	return nil
}
