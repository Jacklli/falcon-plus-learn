package main

import (
	"flag"
	"fmt"
	"github.com/open-falcon/falcon-plus/modules/transfer/g"
	"github.com/open-falcon/falcon-plus/modules/transfer/http"
	"github.com/open-falcon/falcon-plus/modules/transfer/proc"
	"github.com/open-falcon/falcon-plus/modules/transfer/receiver"
	"github.com/open-falcon/falcon-plus/modules/transfer/sender"
	"os"
)

func main() {
	cfg := flag.String("c", "cfg.json", "configuration file")
	version := flag.Bool("v", false, "show version")
	versionGit := flag.Bool("vg", false, "show version")
	flag.Parse()

	if *version {
		fmt.Println(g.VERSION)
		os.Exit(0)
	}
	if *versionGit {
		fmt.Println(g.VERSION, g.COMMIT)
		os.Exit(0)
	}

	// global config
	g.ParseConfig(*cfg) // 加载配置文件到GlobalConfig
	// proc
	proc.Start()

	sender.Start() // 创建连接池、发送队列，初始化一致性哈希，启动发送goroutine
	receiver.Start() // 创建rpcServer和telnetServer，将上报的item放入对应的发送队列

	// http
	http.Start() // 启动httpserver，提供运行状态查询、统计信息查询、上报数据等接口

	select {}
}
