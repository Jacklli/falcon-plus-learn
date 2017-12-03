package sender

import (
	"github.com/open-falcon/falcon-plus/modules/transfer/g"
	nlist "github.com/toolkits/container/list"
)
/*
创建judge、graph、tsdb的发送队列
 */
func initSendQueues() {
	cfg := g.Config()
	/*
	每个node创建一个queue
	        "cluster": {
            "judge-00" : "host0:port0",
            "judge-01" : "host1:port1",
            "judge-02" : "host2:port2",
        }
    共创建3个queue，每个queue承担1/3的数据
	 */
	for node := range cfg.Judge.Cluster {
		Q := nlist.NewSafeListLimited(DefaultSendQueueMaxSize) // 返回&SafeListLimited{SL: &SafeList{L: list.New()}, maxSize: maxSize}
		JudgeQueues[node] = Q
	}

	/*
	每个address对应一个queue
        "cluster": {
            "graph-00" : &ClusterNode{Addrs: ["host0a:port0a", "host0b:port0b"],
            "graph-01" : &ClusterNode{Addrs: ["host1a:port1a", "host1b:port1b"],
            "graph-02" : &ClusterNode{Addrs: ["host2a:port2a", "host2b:port2b"],
        }
    共创建6个queue，graph-00对应两个queue，数据相同，互为备份
	 */
	for node, nitem := range cfg.Graph.ClusterList {
		for _, addr := range nitem.Addrs {
			Q := nlist.NewSafeListLimited(DefaultSendQueueMaxSize) // 返回&SafeListLimited{SL: &SafeList{L: list.New()}, maxSize: maxSize}
			GraphQueues[node+addr] = Q
		}
	}

	if cfg.Tsdb.Enabled {
		TsdbQueue = nlist.NewSafeListLimited(DefaultSendQueueMaxSize) // 返回&SafeListLimited{SL: &SafeList{L: list.New()}, maxSize: maxSize}
	}
}
