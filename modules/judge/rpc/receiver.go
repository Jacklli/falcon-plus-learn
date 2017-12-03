package rpc

import (
	"github.com/open-falcon/falcon-plus/common/model"
	"github.com/open-falcon/falcon-plus/modules/judge/g"
	"github.com/open-falcon/falcon-plus/modules/judge/store"
	"time"
)

type Judge int

func (this *Judge) Ping(req model.NullRpcRequest, resp *model.SimpleRpcResponse) error {
	return nil
}

/*
将transfer上传的metric经过两层hash保存到HistoryBigMap中（旧数据会被清理），
根据g.StrategyMap和g.ExpressionMap进行计算，与阈值比较，判断是否触发报警，
结合之前的event状态，决定是否产生新的event，并将产生的新event保存到g.LastEvents
 */
func (this *Judge) Send(items []*model.JudgeItem, resp *model.SimpleRpcResponse) error {
	remain := g.Config().Remain
	// 把当前时间的计算放在最外层，是为了减少获取时间时的系统调用开销
	now := time.Now().Unix()
	for _, item := range items {
		pk := item.PrimaryKey() // 根据endpoint、metric、tags计算hash key
		store.HistoryBigMap[pk[0:2]].PushFrontAndMaintain(pk, item, remain, now) // 将新上报的metric值插入map，删除旧值只保留固定个数，然后触发judge
	}
	return nil
}
