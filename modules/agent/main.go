package main

import (
	"flag"
	"fmt"
	"github.com/open-falcon/falcon-plus/modules/agent/cron"
	"github.com/open-falcon/falcon-plus/modules/agent/funcs"
	"github.com/open-falcon/falcon-plus/modules/agent/g"
	"github.com/open-falcon/falcon-plus/modules/agent/http"
	"os"
)

/*
配置说明：
{
    "debug": true, # 控制一些debug信息的输出，生产环境通常设置为false
    "hostname": "", # agent采集了数据发给transfer，endpoint就设置为了hostname，默认通过`hostname`获取，如果配置中配置了hostname，就用配置中的
    "ip": "", # agent与hbs心跳的时候会把自己的ip地址发给hbs，agent会自动探测本机ip，如果不想让agent自动探测，可以手工修改该配置
    "plugin": {
        "enabled": false, # 默认不开启插件机制
        "dir": "./plugin", # 把放置插件脚本的git repo clone到这个目录
        "git": "https://github.com/open-falcon/plugin.git", # 放置插件脚本的git repo地址
        "logs": "./logs" # 插件执行的log，如果插件执行有问题，可以去这个目录看log
    },
    "heartbeat": {
        "enabled": true, # 此处enabled要设置为true
        "addr": "127.0.0.1:6030", # hbs的地址，端口是hbs的rpc端口
        "interval": 60, # 心跳周期，单位是秒
        "timeout": 1000 # 连接hbs的超时时间，单位是毫秒
    },
    "transfer": {
        "enabled": true, # 此处enabled要设置为true
        "addrs": [
            "127.0.0.1:8433",
            "127.0.0.1:8433"
        ], # transfer的地址，端口是transfer的rpc端口, 可以支持写多个transfer的地址，agent会保证HA
        "interval": 60, # 采集周期，单位是秒，即agent一分钟采集一次数据发给transfer
        "timeout": 1000 # 连接transfer的超时时间，单位是毫秒
    },
    "http": {
        "enabled": true, # 是否要监听http端口
        "listen": ":1988" # 如果监听的话，监听的地址
    },
    "collector": {
        "ifacePrefix": ["eth", "em"] # 默认配置只会采集网卡名称前缀是eth、em的网卡流量，配置为空就会采集所有的，lo的也会采集。可以从/proc/net/dev看到各个网卡的流量信息
    },
    "ignore": { # 默认采集了200多个metric，可以通过ignore设置为不采集
        "cpu.busy": true,
        "mem.swapfree": true
    }
}
 */

func main() {

	cfg := flag.String("c", "cfg.json", "configuration file")
	version := flag.Bool("v", false, "show version")
	check := flag.Bool("check", false, "check collector")

	// 参数解析
	flag.Parse()

	if *version {
		fmt.Println(g.VERSION)
		os.Exit(0)
	}

	if *check {
		funcs.CheckCollector() // 检查各种metric采集器工作是否正常
		os.Exit(0)
	}

	g.ParseConfig(*cfg) // 加载配置文件到config *GlobalConfig

	if g.Config().Debug { // 设置日志级别log.SetLevel
		g.InitLog("debug")
	} else {
		g.InitLog("info")
	}

	g.InitRootDir() // 获取当前工作目录保存到全局变量Root
	g.InitLocalIp() // 通过连接HBS，获取本机IP，保存到全局变量LocalIp
	g.InitRpcClients() // 创建一个HbsClient *SingleConnRpcClient，这个时候还未连接

	funcs.BuildMappers() // 构造metric采集函数和采集周期列表

	go cron.InitDataHistory() // 定期更新procStatHistory和diskStatsMap，只保留最近两个值

	cron.ReportAgentStatus() // 定期调用rpc method：Agent.ReportStatus，上报agent状态
	cron.SyncMinePlugins() // 定期获取Plugin信息，更新本地Plugin状态，进行调度，Plugin执行结果发送到transfer
	cron.SyncBuiltinMetrics() // 周期性通过rpc调用Agent.BuiltinMetrics获取reportUrls、reportPorts、reportProcs、duPaths检查参数
	cron.SyncTrustableIps() // 周期性通过rpc调用Agent.TrustableIps，获取IP白名单，并设置全局变量ips
	cron.Collect() // 周期性调用funcs.Mappers[i].Fs，进行metric采集，并发送至transfer

	go http.Start() // 启动httpserver，提供查询、操作接口，会判断TrustableIps

	select {}
	// An empty select{} statement blocks indefinitely i.e. forever. It is similar and in practice equivalent to an empty for{} statement.

}
