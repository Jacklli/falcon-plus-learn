package main

import (
	"flag"
	"fmt"
	"github.com/open-falcon/falcon-plus/modules/judge/cron"
	"github.com/open-falcon/falcon-plus/modules/judge/g"
	"github.com/open-falcon/falcon-plus/modules/judge/http"
	"github.com/open-falcon/falcon-plus/modules/judge/rpc"
	"github.com/open-falcon/falcon-plus/modules/judge/store"
	"os"
)

func main() {
	cfg := flag.String("c", "cfg.json", "configuration file")
	version := flag.Bool("v", false, "show version")
	flag.Parse()

	if *version {
		fmt.Println(g.VERSION)
		os.Exit(0)
	}

	g.ParseConfig(*cfg) // 加载配置文件到config *GlobalConfig

	g.InitRedisConnPool() // 初始化redis连接池
	g.InitHbsClient() // 初始化rpc客户端，用于连接hbs

	store.InitHistoryBigMap() // 创建HistoryBigMap，用于存放最近的metrics，用于报警判断

	go http.Start() // 启动http server，在init中注册处理函数configCommonRoutes和configInfoRoutes
	go rpc.Start() // 启动rpc server，接收数据并更新到HistoryBigMap，然后进行judge判断，报警事件写入redis

	go cron.SyncStrategies() // 周期性从hbs下载strategy和expression的配置
	go cron.CleanStale() // 周期性清理近7天没有新数据上报的key

	select {}
}
