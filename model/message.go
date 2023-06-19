package model

import (
	"context"
	"finance/contrib/helper"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	g "github.com/doug-martin/goqu/v9"
	"strconv"
	"time"
)

// 发送站内信
func messageSend(msgID, title, content, sendName, prefix string, isTop, isVip, ty int, names []string) error {

	record := g.Record{
		"message_id": msgID,
		"title":      title,
		"content":    content,
		"send_name":  sendName,
		"is_top":     isTop,
		"is_vip":     isVip,
		"is_read":    0,
		"is_delete":  0,
		"send_at":    time.Now().Unix(),
		"ty":         ty,
	}
	coll := meta.MgDB.Collection("stb_messages")

	var recordList []interface{}
	for k, v := range names {
		t := time.Now().UnixMicro() + int64(k)

		timeNow := time.UnixMicro(t)
		timeString := timeNow.In(loc).Format("2006-01-02T15:04:05.999999+07:00")
		item := g.Record{
			"message_id": record["message_id"],
			"title":      record["title"],
			"content":    record["content"],
			"send_name":  record["send_name"],
			"is_top":     record["is_top"],
			"is_vip":     record["is_vip"],
			"is_read":    record["is_read"],
			"send_at":    record["send_at"],
			"ty":         record["ty"],
		}
		item["ts"] = timeString
		item["username"] = v
		item["prefix"] = prefix
		recordList = append(recordList, item)
	}
	if len(recordList) > 0 {
		pageSize := 200
		page := 1
		if len(recordList) > pageSize {
			for page*pageSize < len(recordList) {
				start := (page - 1) * pageSize
				end := page * pageSize
				pickList := recordList[start:end]
				_, err := coll.InsertMany(ctx, pickList)
				if err != nil {
					fmt.Println("mongo insert error=", err)
					return pushLog(err, helper.DBErr)
				}
				page++
			}
			fmt.Println("-----------")
			start := (page - 1) * pageSize
			end := len(recordList)
			pickList := recordList[start:end]
			fmt.Println(pickList)
			_, err := coll.InsertMany(ctx, pickList)
			if err != nil {
				fmt.Println("mongo insert error=", err)
				return pushLog(err, helper.DBErr)
			}
		} else {
			_, err := coll.InsertMany(ctx, recordList)
			if err != nil {
				fmt.Println("mongo insert error=", err)
				return pushLog(err, helper.DBErr)
			}
		}
	}

	return nil
}

// 写入日志
func paymentPushLog(data paymentTDLog) {

	if data.Error == "" {
		data.Level = "info"
	} else {
		data.Level = "error"
	}

	ts := time.Now()
	l := map[string]string{
		"username":      data.Username,
		"lable":         paymentLogTag,
		"order_id":      data.OrderID,
		"level":         data.Level,
		"error":         data.Error,
		"response_body": data.ResponseBody,
		"response_code": strconv.Itoa(data.ResponseCode),
		"request_body":  data.RequestBody,
		"request_url":   data.RequestURL,
		"merchant":      data.Merchant,
		"channel":       data.Channel,
		"flag":          data.Flag,
		"_index":        fmt.Sprintf("%s_payment_log_%04d%02d", meta.Prefix, ts.Year(), ts.Month()),
	}

	param, _ := helper.JsonMarshal(l)
	err := meta.MerchantMQ.SendAsync(ctx,
		func(c context.Context, result *primitive.SendResult, e error) {
			if e != nil {
				fmt.Printf("receive message error: %s\n", e.Error())
			}
		}, primitive.NewMessage("zinc_fluent_log", param))
	if err != nil {
		fmt.Printf("pushLog %#v\n%s", data, err.Error())
	}
}
