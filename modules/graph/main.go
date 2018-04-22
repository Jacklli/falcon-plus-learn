package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/open-falcon/falcon-plus/modules/graph/api"
	"github.com/open-falcon/falcon-plus/modules/graph/cron"
	"github.com/open-falcon/falcon-plus/modules/graph/g"
	"github.com/open-falcon/falcon-plus/modules/graph/http"
	"github.com/open-falcon/falcon-plus/modules/graph/index"
	"github.com/open-falcon/falcon-plus/modules/graph/rrdtool"
)

/*
注册信号处理，在接收到syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT信号时，
依次关闭http、rpc服务，rrd落盘
 */
func start_signal(pid int, cfg *g.GlobalConfig) {
	sigs := make(chan os.Signal, 1)
	log.Println(pid, "register signal notify")
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		s := <-sigs
		log.Println("recv", s)

		switch s {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			log.Println("graceful shut down")
			if cfg.Http.Enabled {
				http.Close_chan <- 1
				<-http.Close_done_chan
			}
			log.Println("http stop ok")

			if cfg.Rpc.Enabled {
				api.Close_chan <- 1
				<-api.Close_done_chan
			}
			log.Println("rpc stop ok")

			rrdtool.Out_done_chan <- 1
			rrdtool.FlushAll(true)  // 全部刷新到文件
			log.Println("rrdtool stop ok")

			log.Println(pid, "exit")
			os.Exit(0)
		}
	}
}

func main() {
	cfg := flag.String("c", "cfg.json", "specify config file")
	version := flag.Bool("v", false, "show version")
	versionGit := flag.Bool("vg", false, "show version and git commit log")
	flag.Parse()  // 参数解析

	if *version {  // 打印版本信息
		fmt.Println(g.VERSION)
		os.Exit(0)
	}
	if *versionGit {
		fmt.Println(g.VERSION, g.COMMIT)
		os.Exit(0)
	}

	// global config
	g.ParseConfig(*cfg)  // 加载配置文件到GlobalConfig ptr

	if g.Config().Debug {  // 设置日志级别log.SetLevel
		g.InitLog("debug")
	} else {
		g.InitLog("info")
		gin.SetMode(gin.ReleaseMode)
		/*
		Gin is a web framework written in Go (Golang).
		It features a martini-like API with much better performance,
		up to 40 times faster thanks to httprouter.
		If you need performance and good productivity, you will love Gin.

		HttpRouter is a lightweight high performance HTTP request router
		(also called multiplexer or just mux for short) for Go.
		In contrast to the default mux of Go's net/http package,
		this router supports variables in the routing pattern and matches against the request method.
		It also scales better.The router is optimized for high performance and a small memory footprint.
		It scales well even with very long paths and a large number of routes.
		A compressing dynamic trie (radix tree) structure is used for efficient matching.
		 */
	}

	// init db
	g.InitDB() // 创建数据库连接DB *sql.DB，看是否能成功创建，初始化连接池dbConnMap
	// rrdtool before api for disable loopback connection
	/*
	开启net_task_worker goroutine监听net_task任务管道用于同其他graph传输监控数据
	开启ioworker goroutine监听io_task任务管道用于读写本地rrd文件
	开启syncdisk goroutine用于周期性刷新GraphItems缓存到rrd文件
	 */
	rrdtool.Start()
	// start api
	go api.Start() // 启动rpc服务，提供监控数据传输接口
	// start indexing
	index.Start() // 初始化索引功能模块
	// start http server
	go http.Start() // 启动http服务，提供统计信息查询接口
	go cron.CleanCache() // 开启GraphItems和historyCache清理goroutine

	start_signal(os.Getpid(), g.Config()) // 注册信号处理函数，关闭各个goroutine
}
