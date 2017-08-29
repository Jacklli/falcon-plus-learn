package main

import (
	"flag"
	"fmt"
	"github.com/open-falcon/falcon-plus/modules/hbs/cache"
	"github.com/open-falcon/falcon-plus/modules/hbs/db"
	"github.com/open-falcon/falcon-plus/modules/hbs/g"
	"github.com/open-falcon/falcon-plus/modules/hbs/http"
	"github.com/open-falcon/falcon-plus/modules/hbs/rpc"
	"os"
	"os/signal"
	"syscall"
)

/*
配置说明：
{
    "debug": true,
    "database": "root:password@tcp(127.0.0.1:3306)/falcon_portal?loc=Local&parseTime=true", # Portal的数据库地址
    "hosts": "", # portal数据库中有个host表，如果表中数据是从其他系统同步过来的，此处配置为sync，否则就维持默认，留空即可
    "maxIdle": 100,
    "listen": ":6030", # hbs监听的rpc地址
    "trustable": [""],
    "http": {
        "enabled": true,
        "listen": "0.0.0.0:6031" # hbs监听的http地址
    }
}
 */

func main() {
	cfg := flag.String("c", "cfg.json", "configuration file")
	version := flag.Bool("v", false, "show version")
	flag.Parse()  // 解析命令行

	// 打印version信息
	if *version {
		fmt.Println(g.VERSION)
		os.Exit(0)
	}

	g.ParseConfig(*cfg) // 加载配置文件到GlobalConfig

	db.Init() // 初始化数据库连接
	cache.Init() // 周期性加载hostgroup、plugin、host、template、strategy、expression等配置信息到内存

	go cache.DeleteStaleAgents() // 每天运行一次，删除内存中超过一天没有心跳的agent

	go http.Start() // 启动http server
	go rpc.Start() // 启动rpc server

	/*
	程序退出信号处理，golang中的信号处理，请看底部demo
	 */
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		fmt.Println()
		db.DB.Close()
		os.Exit(0)
	}()

	select {}
}

/*
golang中的信号处理
有时候我们想在Go程序中处理Signal信号，比如收到SIGTERM信号后优雅的关闭程序(参看下一节的应用)。
Go信号通知机制可以通过往一个channel中发送os.Signal实现。
首先我们创建一个os.Signal channel，然后使用signal.Notify注册要接收的信号。

package main
import "fmt"
import "os"
import "os/signal"
import "syscall"
func main() {
    // Go signal notification works by sending `os.Signal`
    // values on a channel. We'll create a channel to
    // receive these notifications (we'll also make one to
    // notify us when the program can exit).
    sigs := make(chan os.Signal, 1)
    done := make(chan bool, 1)
    // `signal.Notify` registers the given channel to
    // receive notifications of the specified signals.
    signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
    // This goroutine executes a blocking receive for
    // signals. When it gets one it'll print it out
    // and then notify the program that it can finish.
    go func() {
        sig := <-sigs
        fmt.Println()
        fmt.Println(sig)
        done <- true
    }()
    // The program will wait here until it gets the
    // expected signal (as indicated by the goroutine
    // above sending a value on `done`) and then exit.
    fmt.Println("awaiting signal")
    <-done
    fmt.Println("exiting")
}
 */