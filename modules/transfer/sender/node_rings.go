package sender

import (
	cutils "github.com/open-falcon/falcon-plus/common/utils"
	"github.com/open-falcon/falcon-plus/modules/transfer/g"
	rings "github.com/toolkits/consistent/rings"
)
/*
初始化一致性hash信息，如副本数量、节点信息
一致性哈希请参考：http://blog.csdn.net/cywosp/article/details/23397179/
 */
func initNodeRings() {
	cfg := g.Config()

	JudgeNodeRing = rings.NewConsistentHashNodesRing(int32(cfg.Judge.Replicas), cutils.KeysOfMap(cfg.Judge.Cluster))
	GraphNodeRing = rings.NewConsistentHashNodesRing(int32(cfg.Graph.Replicas), cutils.KeysOfMap(cfg.Graph.Cluster))
}
