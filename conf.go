package main

type conf struct {
	Lang           string   `toml:"lang"`
	SmsPrefix      string   `toml:"sms_prefix"`
	Prefix         string   `toml:"prefix"`
	EsPrefix       string   `toml:"es_prefix"`
	GcsDoamin      string   `toml:"gcs_doamin"`
	PullPrefix     string   `toml:"pull_prefix"`
	IsDev          bool     `toml:"is_dev"`
	AutoCommission bool     `toml:"auto_commission"`
	Sock5          string   `toml:"sock5"`
	RPC            string   `toml:"rpc"`
	Fcallback      string   `toml:"fcallback"`
	IndexURL       string   `toml:"index_url"`
	AutoPayLimit   string   `toml:"autoPayLimit"`
	Total          int      `toml:"total"`
	Over6PullLast  bool     `toml:"over6_pull_last"`
	IpdbPath       string   `toml:"ipdb_path"`
	Rocketmq       []string `toml:"rocketmq"`
	Nats           struct {
		Servers  []string `toml:"servers"`
		Username string   `toml:"username"`
		Password string   `toml:"password"`
	} `toml:"nats"`
	Beanstalkd struct {
		Addr    string `toml:"addr"`
		MaxIdle int    `toml:"maxIdle"`
		MaxCap  int    `toml:"maxCap"`
	} `toml:"beanstalkd"`
	BeanBet struct {
		Addr    string `toml:"addr"`
		MaxIdle int    `toml:"maxIdle"`
		MaxCap  int    `toml:"maxCap"`
	} `toml:"bean_bet"`
	Db struct {
		Masteren struct {
			Addr        string `toml:"addr"`
			MaxIdleConn int    `toml:"max_idle_conn"`
			MaxOpenConn int    `toml:"max_open_conn"`
		} `toml:"masteren"`
		Dorisen struct {
			Addr        string `toml:"addr"`
			MaxIdleConn int    `toml:"max_idle_conn"`
			MaxOpenConn int    `toml:"max_open_conn"`
		} `toml:"dorisen"`
		Beten struct {
			Addr        string `toml:"addr"`
			MaxIdleConn int    `toml:"max_idle_conn"`
			MaxOpenConn int    `toml:"max_open_conn"`
		} `toml:"beten"`
	} `toml:"db"`
	Td struct {
		Log struct {
			Addr        string `toml:"addr"`
			MaxIdleConn int    `toml:"max_idle_conn"`
			MaxOpenConn int    `toml:"max_open_conn"`
		} `toml:"log"`
		Message struct {
			Addr        string `toml:"addr"`
			MaxIdleConn int    `toml:"max_idle_conn"`
			MaxOpenConn int    `toml:"max_open_conn"`
		} `toml:"message"`
		Chat struct {
			Addr        string `toml:"addr"`
			MaxIdleConn int    `toml:"max_idle_conn"`
			MaxOpenConn int    `toml:"max_open_conn"`
		} `toml:"chat"`
	} `toml:"td"`
	BankcardValidAPI struct {
		URL string `toml:"url"`
		Key string `toml:"key"`
	} `toml:"bankcard_valid_api"`
	Fluentd struct {
		Host string `toml:"host"`
		Port int    `toml:"port"`
	} `toml:"fluentd"`
	Redis struct {
		Addr     []string `toml:"addr"`
		Password string   `toml:"password"`
		Sentinel string   `toml:"sentinel"`
		Db       int      `toml:"db"`
	} `toml:"redis"`
	Es struct {
		Host     []string `toml:"host"`
		Username string   `toml:"username"`
		Password string   `toml:"password"`
	} `toml:"es"`
	Redis1 struct {
		Addr     string `toml:"addr"`
		Password string `toml:"password"`
	} `toml:"redis1"`
	AccessEs struct {
		Host     []string `toml:"host"`
		Username string   `toml:"username"`
		Password string   `toml:"password"`
	} `toml:"access_es"`
	Zinc struct {
		URL      string `toml:"url"`
		Username string `toml:"username"`
		Password string `toml:"password"`
	} `toml:"zinc"`
	Port struct {
		Game        string `toml:"game"`
		Member      string `toml:"member"`
		Promo       string `toml:"promo"`
		PromoRPC    string `toml:"promo_rpc"`
		Merchant    string `toml:"merchant"`
		Finance     string `toml:"finance"`
		Sms         string `toml:"sms"`
		Shorturl    string `toml:"shorturl"`
		ShorturlRPC string `toml:"shorturl_rpc"`
		Log         string `toml:"log"`
		IPRPC       string `toml:"ip_rpc"`
	} `toml:"port"`
	Mongodb struct {
		Url      []string `toml:"url"`
		Username string   `toml:"username"`
		Password string   `toml:"password"`
		Db       string   `toml:"db"`
	} `toml:"mongodb"`
}
