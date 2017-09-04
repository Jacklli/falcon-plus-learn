package sender

import (
	backend "github.com/open-falcon/falcon-plus/common/backend_pool"
	"github.com/open-falcon/falcon-plus/modules/transfer/g"
	nset "github.com/toolkits/container/set"
)
/*
创建judge,tsdb,graph的rpc连接池
 */
func initConnPools() {
	cfg := g.Config()

	// judge
	/*
	初始化judge cluster连接池
	        "cluster": {
            "judge-00" : "host0:port0",
            "judge-01" : "host1:port1",
            "judge-02" : "host2:port2",
        }
	 */
	judgeInstances := nset.NewStringSet() // 创建一个set（map[string]struct{}），存储"host:port"，用于去重
	for _, instance := range cfg.Judge.Cluster { // 将Judge的cluster信息加入set
		judgeInstances.Add(instance)
	} // 结果：judgeInstances.M = {"host0:port0":struct{}{},"host1:port1":struct{}{},"host2:port2":struct{}{}}
	/*
	遍历judgeInstances.ToSlice()，即["host0:port0","host1:port1","host2:port2"]
	针对每一个address，创建一个ConnPool，维护在SafeRpcConnPools.M中，返回SafeRpcConnPools地址
	 */
	JudgeConnPools = backend.CreateSafeRpcConnPools(cfg.Judge.MaxConns, cfg.Judge.MaxIdle,
		cfg.Judge.ConnTimeout, cfg.Judge.CallTimeout, judgeInstances.ToSlice())

	// tsdb
	if cfg.Tsdb.Enabled {
		// 创建一个TsdbConnPoolHelper，返回其地址
		TsdbConnPoolHelper = backend.NewTsdbConnPoolHelper(cfg.Tsdb.Address, cfg.Tsdb.MaxConns, cfg.Tsdb.MaxIdle, cfg.Tsdb.ConnTimeout, cfg.Tsdb.CallTimeout)
	}

	// graph
	/*
	初始化graph cluster连接池
	        "cluster": {
            "graph-00" : &ClusterNode{Addrs: ["host0a:port0a", "host0b:port0b"],
            "graph-01" : &ClusterNode{Addrs: ["host1a:port1a", "host1b:port1b"],
            "graph-02" : &ClusterNode{Addrs: ["host2a:port2a", "host2b:port2b"],
        }
	 */
	graphInstances := nset.NewSafeSet() // 创建一个set（map[string]bool），存储"host:port"，用于去重
	for _, nitem := range cfg.Graph.ClusterList { // 将Graph的cluster信息加入set
		for _, addr := range nitem.Addrs {
			graphInstances.Add(addr)
		}
	} // 结果：graphInstances.M = {"host0a:port0a":true,"host0b:port0b":true,"host1a:port1a":true ... }
	/*
	遍历graphInstances.ToSlice()，即["host0a:port0a", "host0b:port0b","host1a:port1a" ...]
	针对每一个address，创建一个ConnPool，维护在SafeRpcConnPools.M中，返回SafeRpcConnPools地址
	 */
	GraphConnPools = backend.CreateSafeRpcConnPools(cfg.Graph.MaxConns, cfg.Graph.MaxIdle,
		cfg.Graph.ConnTimeout, cfg.Graph.CallTimeout, graphInstances.ToSlice())

}

func DestroyConnPools() {
	JudgeConnPools.Destroy()
	GraphConnPools.Destroy()
	TsdbConnPoolHelper.Destroy()
}
