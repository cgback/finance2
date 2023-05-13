package model

import (
	"finance/contrib/helper"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
)

func ChannelTypeList() ([]ChannelType, error) {

	var data []ChannelType
	query, _, _ := dialect.From("tbl_channel_type").Select(colsChannelType...).Where(g.Ex{"state": 1}).ToSQL()
	fmt.Println(query)
	err := meta.MerchantDB.Select(&data, query)
	if err != nil {
		return data, pushLog(fmt.Errorf("%s,[%s]", err.Error(), query), helper.DBErr)
	}

	return data, nil
}
