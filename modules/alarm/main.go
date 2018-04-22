package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/open-falcon/falcon-plus/modules/alarm/cron"
	"github.com/open-falcon/falcon-plus/modules/alarm/g"
	"github.com/open-falcon/falcon-plus/modules/alarm/http"
	"github.com/open-falcon/falcon-plus/modules/alarm/model"
)

func main() {
	cfg := flag.String("c", "cfg.json", "configuration file")
	version := flag.Bool("v", false, "show version")
	help := flag.Bool("h", false, "help")
	flag.Parse()

	if *version {
		fmt.Println(g.VERSION)
		os.Exit(0)
	}

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	g.ParseConfig(*cfg) // 加载配置文件

	g.InitLog(g.Config().LogLevel) // 根据配置文件设置日志级别
	if g.Config().LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
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

	g.InitRedisConnPool() // 创建redis连接池
	model.InitDatabase() // 创建数据库连接和数据库表，使用了beego的orm
	cron.InitSenderWorker() // 创建IMWorkerChan/SmsWorkerChan/MailWorkerChan，根据配置设置channel长度
	                        // 实际在用信号量做并发控制，避免同时发送过多

	go http.Start() // 启动http服务（使用gin框架实现），提供版本查询、健康查询、工作查询接口
	go cron.ReadHighEvent() // 处理高优先级报警事件，包括入库、调callback、报警入redis待发送队列
	go cron.ReadLowEvent() // 处理低优先级报警事件，包括入库、调callback、报警入redis待合并队列
	go cron.CombineSms() // 读取/queue/user/sms队列中的短信内容，聚合后入库，并发送聚合短信
	go cron.CombineMail() // 读取/queue/user/mail队列中的邮件内容，聚合发送
	go cron.CombineIM() // 读取/queue/user/im队列中的IM内容，聚合后入库，并发送聚合消息
	go cron.ConsumeIM() // 读取/im队列中的内容，调用微信发送网关地址发送
	go cron.ConsumeSms() // 读取/sms队列中的内容，调用短信发送网关地址发送
	go cron.ConsumeMail() // 读取/mail队列中的内容，调用邮件发送网关地址发送
	go cron.CleanExpiredEvent() // 删除events表的旧数据，默认7天之前

	// 注册信号处理程序，监听syscall.SIGINT和syscall.SIGTERM信号
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		fmt.Println()
		g.RedisConnPool.Close()
		os.Exit(0)
	}()

	select {} // 等待
}
