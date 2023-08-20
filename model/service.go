package model

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TelegramBotNotice(program, username, gitReversion, buildTime, buildGoVersion, flag, prefix, proxy string, env, botID string, noticeGroupID int64) {

	var localIp string
	ts := time.Now()
	httpClient := &http.Client{}
	//fmt.Println("proxy:", proxy)
	if len(proxy) > 0 && proxy != "0.0.0.0" {
		proxyUrl, _ := url.Parse(fmt.Sprintf(`http://%s`, proxy))
		httpClient = &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}
		fmt.Println("使用了代理")
	}
	bot, err := tgbotapi.NewBotAPIWithClient(botID, tgbotapi.APIEndpoint, httpClient)
	if err != nil {
		fmt.Printf("bot init error: %s\n", err.Error())
	}

	bot.Debug = false
	format := "‼️‼️%s服務啟動‼️‼️\r\n✅✅✅✅✅✅️\r\n⚠️Env: \t%s\r\n⚠️Datetime: \t%s\r\n⚠️Username: \t%s\r\n⚠️GitReversion: \t%s\r\n⚠️BuildTime: \t%s\r\n⚠️BuildGoVersion: \t%s\r\n⚠️Hostname: \t%s\r\n⚠️IP: \t%s\r\n⚠️Flag: \t%s\r\n⚠️Prefix: \t%s\n✨ ✨ ✨ ✨ ✨ ✨\r\n"
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
		return
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatal(err)
	}

	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				localIp = ipnet.IP.String()
				break
			}
		}
	}

	msg := tgbotapi.NewMessage(noticeGroupID, "")
	msg.Text = fmt.Sprintf(format, program, env, ts.Format("2006-01-02 15:04:05"), username, gitReversion, buildTime, buildGoVersion, hostname, localIp, flag, prefix)
	if _, err := bot.Send(msg); err != nil {
		fmt.Println("tgbot error : ", err.Error())
	}
}
