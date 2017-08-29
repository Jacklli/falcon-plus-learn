package rpc

import (
	"fmt"
	"github.com/open-falcon/falcon-plus/common/model"
	"github.com/open-falcon/falcon-plus/common/utils"
	"github.com/open-falcon/falcon-plus/modules/hbs/cache"
)

func (t *Hbs) GetExpressions(req model.NullRpcRequest, reply *model.ExpressionResponse) error {
	reply.Expressions = cache.ExpressionCache.Get()
	return nil
}
/*
返回hostname对应策略列表的全量map
 */
func (t *Hbs) GetStrategies(req model.NullRpcRequest, reply *model.StrategiesResponse) error {
	reply.HostStrategies = []*model.HostStrategy{}
	// 一个机器ID对应多个模板ID
	hidTids := cache.HostTemplateIds.GetMap()
	sz := len(hidTids)
	if sz == 0 {
		return nil
	}

	// Judge需要的是hostname，此处要把HostId转换为hostname
	// 查出的hosts，是不处于维护时间内的
	hosts := cache.MonitoredHosts.Get()
	if len(hosts) == 0 {
		// 所有机器都处于维护状态，汗
		return nil
	}

	tpls := cache.TemplateCache.GetMap() // 查询所有template信息
	if len(tpls) == 0 {
		return nil
	}

	strategies := cache.Strategies.GetMap() // 查询所有active的strategy信息
	if len(strategies) == 0 {
		return nil
	}

	// 做个索引，给一个tplId，可以很方便的找到对应了哪些Strategy
	tpl2Strategies := Tpl2Strategies(strategies)

	hostStrategies := make([]*model.HostStrategy, 0, sz)
	for hostId, tplIds := range hidTids {

		h, exists := hosts[hostId]
		if !exists {
			continue
		}

		// 计算当前host配置了哪些监控策略
		ss := CalcInheritStrategies(tpls, tplIds, tpl2Strategies)
		if len(ss) <= 0 {
			continue
		}

		hs := model.HostStrategy{
			Hostname:   h.Name,
			Strategies: ss,
		}

		hostStrategies = append(hostStrategies, &hs)

	}

	reply.HostStrategies = hostStrategies
	return nil
}
/*
将结构
{
  Strategyid1: &model.Strategy{Tpl: Tpl1},
  Strategyid2: &model.Strategy{Tpl: Tpl1},
  Strategyid3: &model.Strategy{Tpl: Tpl2},
}

转换成：
{
  Tplid1:[]*model.Strategy{Strategy1, Strategy2},
  Tplid2:[]*model.Strategy{Strategy3},
}
 */
func Tpl2Strategies(strategies map[int]*model.Strategy) map[int][]*model.Strategy {
	ret := make(map[int][]*model.Strategy)
	for _, s := range strategies {
		if s == nil || s.Tpl == nil {
			continue
		}
		if _, exists := ret[s.Tpl.Id]; exists {
			ret[s.Tpl.Id] = append(ret[s.Tpl.Id], s)
		} else {
			ret[s.Tpl.Id] = []*model.Strategy{s}
		}
	}
	return ret
}

func CalcInheritStrategies(allTpls map[int]*model.Template, tids []int, tpl2Strategies map[int][]*model.Strategy) []model.Strategy {
	// 根据模板的继承关系，找到每个机器对应的模板全量
	/**
	 * host_id =>
	 * |a |d |a |a |a |
	 * |  |  |b |b |f |
	 * |  |  |  |c |  |
	 * |  |  |  |  |  |
	 */
	tpl_buckets := [][]int{}
	for _, tid := range tids {
		ids := cache.ParentIds(allTpls, tid) // 获取tid的parentid列表，如[tid, pid1, pid2...]
		if len(ids) <= 0 {
			continue
		}
		tpl_buckets = append(tpl_buckets, ids)
	}

	// 每个host 关联的模板，有继承关系的放到同一个bucket中，其他的放在各自单独的bucket中
	/**
	 * host_id =>
	 * |a |d |a |
	 * |b |  |f |
	 * |c |  |  |
	 * |  |  |  |
	 */
	/*
	去重，保留最长的slice，即最顶层的模板
	 */
	uniq_tpl_buckets := [][]int{}
	for i := 0; i < len(tpl_buckets); i++ {
		var valid bool = true
		for j := 0; j < len(tpl_buckets); j++ {
			if i == j {
				continue
			}
			if slice_int_eq(tpl_buckets[i], tpl_buckets[j]) { // 这种情况应该不会出现??如果出现，则buckets会存在重复的模板
				break
			}
			if slice_int_lt(tpl_buckets[i], tpl_buckets[j]) {
				valid = false
				break
			}
		}
		if valid {
			uniq_tpl_buckets = append(uniq_tpl_buckets, tpl_buckets[i])
		}
	}

	// 继承覆盖父模板策略，得到每个模板聚合后的策略列表
	strategies := []model.Strategy{}

	exists_by_id := make(map[int]struct{})
	for _, bucket := range uniq_tpl_buckets {

		// 开始计算一个桶，先计算老的tid，再计算新的，所以可以覆盖
		// 该桶最终结果
		bucket_stras_map := make(map[string][]*model.Strategy) // metric -> []*model.Strategy
		for _, tid := range bucket {

			// 一个tid对应的策略列表
			the_tid_stras := make(map[string][]*model.Strategy)

			if stras, ok := tpl2Strategies[tid]; ok {
				for _, s := range stras {
					uuid := fmt.Sprintf("metric:%s/tags:%v", s.Metric, utils.SortedTags(s.Tags)) // 将tags按key排序，{a:1, c:2, b:3} -> "a=1,b=2,c=3"
					if _, ok2 := the_tid_stras[uuid]; ok2 {
						the_tid_stras[uuid] = append(the_tid_stras[uuid], s)
					} else {
						the_tid_stras[uuid] = []*model.Strategy{s}
					}
				}
			}

			// 覆盖父模板
			for uuid, ss := range the_tid_stras {
				bucket_stras_map[uuid] = ss
			}
		}

		last_tid := bucket[len(bucket)-1]

		// 替换所有策略的模板为最年轻的模板
		for _, ss := range bucket_stras_map {
			for _, s := range ss {
				valStrategy := *s
				// exists_by_id[s.Id] 是根据策略ID去重，不太确定是否真的需要，不过加上肯定没问题
				if _, exist := exists_by_id[valStrategy.Id]; !exist {
					if valStrategy.Tpl.Id != last_tid {
						valStrategy.Tpl = allTpls[last_tid]
					}
					strategies = append(strategies, valStrategy)
					exists_by_id[valStrategy.Id] = struct{}{}
				}
			}
		}
	}

	return strategies
}
/*
如果target在list中，则返回true
 */
func slice_int_contains(list []int, target int) bool {
	for _, b := range list {
		if b == target {
			return true
		}
	}
	return false
}
/*
比较两个slice是否相等，相等放回true
 */
func slice_int_eq(a []int, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i, av := range a {
		if av != b[i] {
			return false
		}
	}
	return true
}
/*
如果slice a的元素都包含在slice b中，则返回true
 */
func slice_int_lt(a []int, b []int) bool {
	for _, i := range a {
		if !slice_int_contains(b, i) {
			return false
		}
	}
	return true
}
