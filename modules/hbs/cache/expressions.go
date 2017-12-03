package cache

import (
	"github.com/open-falcon/falcon-plus/common/model"
	"github.com/open-falcon/falcon-plus/modules/hbs/db"
	"sync"
)

type SafeExpressionCache struct {
	sync.RWMutex
	L []*model.Expression
}

var ExpressionCache = &SafeExpressionCache{}

func (this *SafeExpressionCache) Get() []*model.Expression {
	this.RLock()
	defer this.RUnlock()
	return this.L
}
/*
查询所有active的Expression，保存到ExpressionCache.L
 */
func (this *SafeExpressionCache) Init() {
	es, err := db.QueryExpressions() // 查询所有active的Expression
	if err != nil {
		return
	}

	this.Lock()
	defer this.Unlock()
	this.L = es
}
