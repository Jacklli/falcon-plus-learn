package sender

import (
	"bytes"
	cmodel "github.com/open-falcon/falcon-plus/common/model"
	"github.com/open-falcon/falcon-plus/modules/transfer/g"
	"github.com/open-falcon/falcon-plus/modules/transfer/proc"
	nsema "github.com/toolkits/concurrent/semaphore"
	"github.com/toolkits/container/list"
	"log"
	"time"
)

// send
const (
	DefaultSendTaskSleepInterval = time.Millisecond * 50 //默认睡眠间隔为50ms
)
/*
开启goroutine: forward2JudgeTask、forward2GraphTask、forward2TsdbTask发送SendQueue中的数据
 */
// TODO 添加对发送任务的控制,比如stop等
func startSendTasks() {
	cfg := g.Config()
	// init semaphore
	judgeConcurrent := cfg.Judge.MaxConns
	graphConcurrent := cfg.Graph.MaxConns
	tsdbConcurrent := cfg.Tsdb.MaxConns

	if tsdbConcurrent < 1 {
		tsdbConcurrent = 1
	}

	if judgeConcurrent < 1 {
		judgeConcurrent = 1
	}

	if graphConcurrent < 1 {
		graphConcurrent = 1
	}

	// init send go-routines
	for node := range cfg.Judge.Cluster {
		queue := JudgeQueues[node] // 获取专属发送queue
		go forward2JudgeTask(queue, node, judgeConcurrent) // 从发送queue获取数据，进行批量同步发送，使用信号量控制并发
	}

	for node, nitem := range cfg.Graph.ClusterList {
		for _, addr := range nitem.Addrs {
			queue := GraphQueues[node+addr]
			go forward2GraphTask(queue, node, addr, graphConcurrent) // 从发送queue获取数据，进行批量同步发送，使用信号量控制并发
		}
	}

	if cfg.Tsdb.Enabled {
		go forward2TsdbTask(tsdbConcurrent) // 从发送queue获取数据，进行批量同步发送，使用信号量控制并发
	}
}
/*
从发送queue获取数据，进行批量同步发送，使用信号量控制并发
 */
// Judge定时任务, 将 Judge发送缓存中的数据 通过rpc连接池 发送到Judge
func forward2JudgeTask(Q *list.SafeListLimited, node string, concurrent int) {
	batch := g.Config().Judge.Batch // 一次发送,最多batch条数据
	addr := g.Config().Judge.Cluster[node] // node对应的地址
	sema := nsema.NewSemaphore(concurrent) // 创建信号量，使用管道模拟的

	for {
		items := Q.PopBackBy(batch) // 一次取出最多batch个item，以slice的方式返回
		count := len(items)
		if count == 0 {
			time.Sleep(DefaultSendTaskSleepInterval)
			continue
		}

		// 构造待发送的judgeItems，这里用到了Type assertions
		judgeItems := make([]*cmodel.JudgeItem, count)
		for i := 0; i < count; i++ {
			judgeItems[i] = items[i].(*cmodel.JudgeItem)
		}

		/*
		Type assertions

		A type assertion provides access to an interface value's underlying concrete value.

		t := i.(T)

		This statement asserts that the interface value i holds the concrete type T and assigns the underlying T value to the variable t.

		If i does not hold a T, the statement will trigger a panic.

		To test whether an interface value holds a specific type, a type assertion can return two values: the underlying value and a boolean value that reports whether the assertion succeeded.

		t, ok := i.(T)

		If i holds a T, then t will be the underlying value and ok will be true.

		If not, ok will be false and t will be the zero value of type T, and no panic occurs.

		Note the similarity between this syntax and that of reading from a map.
		*/

		//	同步Call + 有限并发 进行发送
		sema.Acquire() // 通过获取信号量控制并发
		go func(addr string, judgeItems []*cmodel.JudgeItem, count int) {
			defer sema.Release()

			resp := &cmodel.SimpleRpcResponse{}
			var err error
			sendOk := false
			for i := 0; i < 3; i++ { //最多重试3次
				err = JudgeConnPools.Call(addr, "Judge.Send", judgeItems, resp) // 从ConnPool中获取一个连接进行发送
				if err == nil {
					sendOk = true
					break
				}
				time.Sleep(time.Millisecond * 10)
			}

			// 更新统计信息
			// statistics
			if !sendOk {
				log.Printf("send judge %s:%s fail: %v", node, addr, err)
				proc.SendToJudgeFailCnt.IncrBy(int64(count))
			} else {
				proc.SendToJudgeCnt.IncrBy(int64(count))
			}
		}(addr, judgeItems, count)
	}
}

// Graph定时任务, 将 Graph发送缓存中的数据 通过rpc连接池 发送到Graph
func forward2GraphTask(Q *list.SafeListLimited, node string, addr string, concurrent int) {
	batch := g.Config().Graph.Batch // 一次发送,最多batch条数据
	sema := nsema.NewSemaphore(concurrent)

	for {
		items := Q.PopBackBy(batch)
		count := len(items)
		if count == 0 {
			time.Sleep(DefaultSendTaskSleepInterval)
			continue
		}

		graphItems := make([]*cmodel.GraphItem, count)
		for i := 0; i < count; i++ {
			graphItems[i] = items[i].(*cmodel.GraphItem)
		}

		sema.Acquire()
		go func(addr string, graphItems []*cmodel.GraphItem, count int) {
			defer sema.Release()

			resp := &cmodel.SimpleRpcResponse{}
			var err error
			sendOk := false
			for i := 0; i < 3; i++ { //最多重试3次
				err = GraphConnPools.Call(addr, "Graph.Send", graphItems, resp)
				if err == nil {
					sendOk = true
					break
				}
				time.Sleep(time.Millisecond * 10)
			}

			// statistics
			if !sendOk {
				log.Printf("send to graph %s:%s fail: %v", node, addr, err)
				proc.SendToGraphFailCnt.IncrBy(int64(count))
			} else {
				proc.SendToGraphCnt.IncrBy(int64(count))
			}
		}(addr, graphItems, count)
	}
}
/*
从发送queue获取数据，进行批量同步发送，使用信号量控制并发
 */
// Tsdb定时任务, 将数据通过api发送到tsdb
func forward2TsdbTask(concurrent int) {
	batch := g.Config().Tsdb.Batch // 一次发送,最多batch条数据
	retry := g.Config().Tsdb.MaxRetry
	sema := nsema.NewSemaphore(concurrent)

	for {
		items := TsdbQueue.PopBackBy(batch)
		if len(items) == 0 {
			time.Sleep(DefaultSendTaskSleepInterval)
			continue
		}
		//  同步Call + 有限并发 进行发送
		sema.Acquire()
		go func(itemList []interface{}) {
			defer sema.Release()

			// 将itemList转换成string
			var tsdbBuffer bytes.Buffer
			for i := 0; i < len(itemList); i++ {
				tsdbItem := itemList[i].(*cmodel.TsdbItem)
				tsdbBuffer.WriteString(tsdbItem.TsdbString())
				tsdbBuffer.WriteString("\n")
			}

			var err error
			for i := 0; i < retry; i++ {
				err = TsdbConnPoolHelper.Send(tsdbBuffer.Bytes()) // 从TsdbConnPoolHelper.p获取一个可用连进行发送
				if err == nil {
					proc.SendToTsdbCnt.IncrBy(int64(len(itemList)))
					break
				}
				time.Sleep(100 * time.Millisecond)
			}

			if err != nil {
				proc.SendToTsdbFailCnt.IncrBy(int64(len(itemList)))
				log.Println(err)
				return
			}
		}(items)
	}
}
