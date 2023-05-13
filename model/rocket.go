package model

import (
	"context"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/valyala/fasthttp"
)

func rocketPutDelay(topic string, args map[string]string, delayLevel int) error {

	m := &fasthttp.Args{}
	for k, v := range args {
		m.Set(k, v)
	}

	topic = meta.Prefix + "_" + topic
	msg := primitive.NewMessage(topic, m.QueryString())
	msg.WithDelayTimeLevel(delayLevel)
	err := meta.MerchantMQ.SendAsync(ctx,
		func(c context.Context, result *primitive.SendResult, e error) {
			if e != nil {
				fmt.Printf("receive message error: %s\n", e.Error())
			} else {
				fmt.Printf("send message success: result=%s\n", result.String())
			}
		}, msg)

	if err != nil {
		fmt.Printf("send message error: %s\n", err)
	}

	fmt.Println("rocket topic = ", topic)
	fmt.Println("rocket param = ", m.String())
	fmt.Println("rocket err = ", err)

	return err
}

func RocketSendAsync(topic string, payload []byte) error {

	err := meta.MerchantMQ.SendAsync(ctx,
		func(c context.Context, result *primitive.SendResult, e error) {
			if e != nil {
				fmt.Printf("send message error: %s\n", e.Error())
			}
		}, primitive.NewMessage(topic, payload))
	if err != nil {
		fmt.Printf("rocket SendAsync payload[%s] error[%s]", string(payload), err.Error())
	}

	return err
}
