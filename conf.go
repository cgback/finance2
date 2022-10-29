package main

type conf struct {
	Lang     string   `toml:"lang"`
	Prefix   string   `toml:"prefix"`
	Rocketmq []string `toml:"rocketmq"`
	Maindb   struct {
		Addr        string `toml:"addr"`
		MaxIdleConn int    `toml:"max_idle_conn"`
		MaxOpenConn int    `toml:"max_open_conn"`
	} `toml:"maindb"`
	Pika struct {
		Addr     []string `toml:"addr"`
		Password string   `toml:"password"`
		Sentinel string   `toml:"sentinel"`
		Db       int      `toml:"db"`
	} `toml:"pika"`
	Port string `toml:"port"`
}
