package model

import (
	"fmt"
	"github.com/lucacasonato/mqtt"
	"strings"
	"time"
)

func Publish(name string, data []byte) error {

	err := meta.MerchantMqtt.Publish(ctx, name, data, mqtt.AtLeastOnce)
	if err != nil {
		fmt.Printf("merchantNats.Publish %s = %s\n", name, err.Error())

	}

	return err
}

func PushMerchantNotify(format, applyName, username, amount string) error {

	msg := fmt.Sprintf(format, applyName, username, amount, applyName, username, amount, applyName, username, amount)
	msg = strings.TrimSpace(msg)

	topic := fmt.Sprintf("%s/merchant", meta.Prefix)
	err := meta.MerchantMqtt.Publish(ctx, topic, []byte(msg), mqtt.AtLeastOnce)
	if err != nil {
		fmt.Println("merchantNats.Publish finance = ", err.Error())
		return err
	}

	return nil
}

func PushWithdrawNotify(format, username, amount string) error {

	ts := time.Now()
	msg := fmt.Sprintf(format, username, amount, username, amount, username, amount)
	msg = strings.TrimSpace(msg)

	topic := fmt.Sprintf("%s/merchant", meta.Prefix)
	err := meta.MerchantMqtt.Publish(ctx, topic, []byte(msg), mqtt.AtLeastOnce)
	if err != nil {
		fmt.Println("failed", time.Since(ts), err.Error())
		fmt.Println("merchantNats.Publish finance = ", err.Error())
		return err
	}

	fmt.Println("success", time.Since(ts))

	return nil
}
