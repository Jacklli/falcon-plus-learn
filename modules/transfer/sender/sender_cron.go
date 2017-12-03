package sender

import (
	"github.com/open-falcon/falcon-plus/modules/transfer/proc"
	"github.com/toolkits/container/list"
	"log"
	"strings"
	"time"
)

const (
	DefaultProcCronPeriod = time.Duration(5) * time.Second    //ProcCron的周期,默认1s
	DefaultLogCronPeriod  = time.Duration(3600) * time.Second //LogCron的周期,默认300s
)

// send_cron程序入口
func startSenderCron() {
	go startProcCron() // 更新JudgeQueuesCnt和GraphQueuesCnt的统计信息
	go startLogCron() // 打印GraphConnPools的统计信息
}

func startProcCron() {
	for {
		time.Sleep(DefaultProcCronPeriod)
		refreshSendingCacheSize() // 设置JudgeQueuesCnt和GraphQueuesCnt的统计信息
	}
}

func startLogCron() {
	for {
		time.Sleep(DefaultLogCronPeriod)
		logConnPoolsProc() // 打印GraphConnPools的统计信息
	}
}
/*
设置JudgeQueuesCnt和GraphQueuesCnt的统计信息
 */
func refreshSendingCacheSize() {
	proc.JudgeQueuesCnt.SetCnt(calcSendCacheSize(JudgeQueues))
	proc.GraphQueuesCnt.SetCnt(calcSendCacheSize(GraphQueues))
}
/*
计算JudgeQueues的item总数
 */
func calcSendCacheSize(mapList map[string]*list.SafeListLimited) int64 {
	var cnt int64 = 0
	for _, list := range mapList {
		if list != nil {
			cnt += int64(list.Len())
		}
	}
	return cnt
}
/*
打印GraphConnPools的统计信息
 */
func logConnPoolsProc() {
	log.Printf("connPools proc: \n%v", strings.Join(GraphConnPools.Proc(), "\n"))
}
