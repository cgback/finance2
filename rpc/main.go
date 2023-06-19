package ryrpc

import (
	"time"

	rycli "github.com/ryrpc/client"
)

var client *rycli.Client

func Constructor(uri string) {

	client = rycli.NewClient()

	client.SetBaseURL(uri)
	client.SetClientTimeout(12 * time.Second)
}
