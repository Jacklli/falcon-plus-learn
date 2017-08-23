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

	go cron.InitDataHistory() // 定期更新procStatHistory和diskStatsMap，只保留两个值

	cron.ReportAgentStatus() // 定期调用rpc method：Agent.ReportStatus，上报agent状态
	cron.SyncMinePlugins() // 定期获取Plugin信息，更新本地Plugin状态，进行调度，Plugin执行结果发送到transfer
	cron.SyncBuiltinMetrics() // 周期性通过rpc调用Agent.BuiltinMetrics获取reportUrls、reportPorts、reportProcs、duPaths检查参数
	cron.SyncTrustableIps() // 周期性通过rpc调用Agent.TrustableIps，获取IP白名单，并设置全局变量ips
	cron.Collect() // 周期性调用funcs.Mappers[i].Fs，进行metric采集，并发送至transfer

	go http.Start() // 启动httpserver，提供查询、操作接口，会判断TrustableIps

	select {}

}
