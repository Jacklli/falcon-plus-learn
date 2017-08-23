package funcs

import (
	"fmt"
	"github.com/toolkits/nux"
	"github.com/toolkits/sys"
)
/*
检查各种metric采集器工作是否正常
 */
func CheckCollector() {

	output := make(map[string]bool)

	_, procStatErr := nux.CurrentProcStat() // 解析/proc/stat，填充*ProcStat
	_, listDiskErr := nux.ListDiskStats() // 解析/proc/diskstats，填充[]*DiskStats
	ports, listeningPortsErr := nux.ListeningPorts() // 调用sh -c 'ss -t -l -n'获取监听端口（去重）
	procs, psErr := nux.AllProcs() // 遍历/proc/<pid>，填充[]*Proc

	_, duErr := sys.CmdOut("du", "--help")

	output["kernel  "] = len(KernelMetrics()) > 0
	output["df.bytes"] = DeviceMetricsCheck()
	output["net.if  "] = len(CoreNetMetrics([]string{})) > 0
	output["loadavg "] = len(LoadAvgMetrics()) > 0
	output["cpustat "] = procStatErr == nil
	output["disk.io "] = listDiskErr == nil
	output["memory  "] = len(MemMetrics()) > 0
	output["netstat "] = len(NetstatMetrics()) > 0
	output["ss -s   "] = len(SocketStatSummaryMetrics()) > 0
	output["ss -tln "] = listeningPortsErr == nil && len(ports) > 0
	output["ps aux  "] = psErr == nil && len(procs) > 0
	output["du -bs  "] = duErr == nil

	for k, v := range output {
		status := "fail"
		if v {
			status = "ok"
		}
		fmt.Println(k, "...", status)
	}
}
