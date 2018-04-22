package index

import (
	"database/sql"
	log "github.com/Sirupsen/logrus"
	"time"

	nsema "github.com/toolkits/concurrent/semaphore"
	ntime "github.com/toolkits/time"

	"github.com/open-falcon/falcon-plus/modules/graph/g"
	proc "github.com/open-falcon/falcon-plus/modules/graph/proc"
)

const (
	IndexUpdateIncrTaskSleepInterval = time.Duration(1) * time.Second // 增量更新间隔时间, 默认30s
)

var (
	semaUpdateIndexIncr = nsema.NewSemaphore(2) // 索引增量更新时操作mysql的并发控制
)

// 启动索引的 异步、增量更新 任务, 每隔一定时间，刷新cache中的数据到数据库中
func StartIndexUpdateIncrTask() {
	for {
		time.Sleep(IndexUpdateIncrTaskSleepInterval)
		startTs := time.Now().Unix()
		cnt := updateIndexIncr()
		endTs := time.Now().Unix()
		// statistics
		proc.IndexUpdateIncrCnt.SetCnt(int64(cnt))
		proc.IndexUpdateIncr.Incr()
		proc.IndexUpdateIncr.PutOther("lastStartTs", ntime.FormatTs(startTs))
		proc.IndexUpdateIncr.PutOther("lastTimeConsumingInSec", endTs-startTs)
	}
}

// 进行一次增量更新
func updateIndexIncr() int {
	ret := 0
	if unIndexedItemCache == nil || unIndexedItemCache.Size() <= 0 {
		return ret
	}

	dbConn, err := g.GetDbConn("UpdateIndexIncrTask") // 获取一个数据库连接
	if err != nil {
		log.Error("[ERROR] get dbConn fail", err)
		return ret
	}

	// 将unIndexedItemCache中的item更新到mysql，同时添加到IndexedItemCache，并从unIndexedItemCache中删除
	keys := unIndexedItemCache.Keys()
	for _, key := range keys {
		icitem := unIndexedItemCache.Get(key)
		unIndexedItemCache.Remove(key)
		if icitem != nil {
			// 并发更新mysql
			semaUpdateIndexIncr.Acquire()  // 并发量控制
			go func(key string, icitem *IndexCacheItem, dbConn *sql.DB) {
				defer semaUpdateIndexIncr.Release()
				err := updateIndexFromOneItem(icitem.Item, dbConn) // 根据item，更新mysql，包括endpoint、tag_endpoint、endpoint_counter表
				if err != nil {
					proc.IndexUpdateIncrErrorCnt.Incr()
				} else {
					IndexedItemCache.Put(key, icitem)
				}
			}(key, icitem.(*IndexCacheItem), dbConn)
			ret++
		}
	}

	return ret
}
