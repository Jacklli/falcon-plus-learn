package cache

import (
	"github.com/open-falcon/falcon-plus/common/model"
	"github.com/open-falcon/falcon-plus/modules/hbs/db"
	"sync"
)

// 一个HostGroup对应多个Template
type SafeGroupTemplates struct {
	sync.RWMutex
	M map[int][]int
}

var GroupTemplates = &SafeGroupTemplates{M: make(map[int][]int)}

func (this *SafeGroupTemplates) GetTemplateIds(gid int) ([]int, bool) {
	this.RLock()
	defer this.RUnlock()
	templateIds, exists := this.M[gid]
	return templateIds, exists
}
/*
查询hostgroup id对应的template id，保存到GroupTemplates.M
 */
func (this *SafeGroupTemplates) Init() {
	m, err := db.QueryGroupTemplates() // 查询hostgroup id对应的template id
	if err != nil {
		return
	}

	this.Lock()
	defer this.Unlock()
	this.M = m
}

type SafeTemplateCache struct {
	sync.RWMutex
	M map[int]*model.Template
}

var TemplateCache = &SafeTemplateCache{M: make(map[int]*model.Template)}
/*
返回缓存的TemplateCache.M
 */
func (this *SafeTemplateCache) GetMap() map[int]*model.Template {
	this.RLock()
	defer this.RUnlock()
	return this.M
}
/*
查询所有template信息，保存到TemplateCache.M
 */
func (this *SafeTemplateCache) Init() {
	ts, err := db.QueryTemplates() // 查询所有template信息
	if err != nil {
		return
	}

	this.Lock()
	defer this.Unlock()
	this.M = ts
}

type SafeHostTemplateIds struct {
	sync.RWMutex
	M map[int][]int
}

var HostTemplateIds = &SafeHostTemplateIds{M: make(map[int][]int)}

func (this *SafeHostTemplateIds) GetMap() map[int][]int {
	this.RLock()
	defer this.RUnlock()
	return this.M
}
/*
查询host id对应的template id，保存到HostTemplateIds.M
 */
func (this *SafeHostTemplateIds) Init() {
	m, err := db.QueryHostTemplateIds() // 查询host id对应的template id
	if err != nil {
		return
	}

	this.Lock()
	defer this.Unlock()
	this.M = m
}
