package store

import (
	"container/list"
	"errors"
	"hash/crc32"
	"log"
	"sync"

	cmodel "github.com/open-falcon/falcon-plus/common/model"

	"github.com/open-falcon/falcon-plus/modules/graph/g"
)

var GraphItems *GraphItemMap

type GraphItemMap struct {
	sync.RWMutex
	A    []map[string]*SafeLinkedList
	Size int
}

/*
返回key对应的SafeLinkedList
 */
func (this *GraphItemMap) Get(key string) (*SafeLinkedList, bool) {
	this.RLock()
	defer this.RUnlock()
	idx := hashKey(key) % uint32(this.Size)
	val, ok := this.A[idx][key]
	return val, ok
}

// Remove method remove key from GraphItemMap, return true if exists
func (this *GraphItemMap) Remove(key string) bool {
	this.Lock()
	defer this.Unlock()
	idx := hashKey(key) % uint32(this.Size)
	_, exists := this.A[idx][key]
	if !exists {
		return false
	}

	delete(this.A[idx], key)
	return true
}

func (this *GraphItemMap) Getitems(idx int) map[string]*SafeLinkedList {
	this.RLock()
	defer this.RUnlock()
	items := this.A[idx]
	this.A[idx] = make(map[string]*SafeLinkedList)
	return items
}

func (this *GraphItemMap) Set(key string, val *SafeLinkedList) {
	this.Lock()
	defer this.Unlock()
	idx := hashKey(key) % uint32(this.Size)
	this.A[idx][key] = val
}

func (this *GraphItemMap) Len() int {
	this.RLock()
	defer this.RUnlock()
	var l int
	for i := 0; i < this.Size; i++ {
		l += len(this.A[i])
	}
	return l
}

/*
返回key的第一个GraphItem，即最新的item
 */
func (this *GraphItemMap) First(key string) *cmodel.GraphItem {
	this.RLock()
	defer this.RUnlock()
	idx := hashKey(key) % uint32(this.Size)
	sl, ok := this.A[idx][key]
	if !ok {
		return nil
	}

	first := sl.Front()
	if first == nil {
		return nil
	}

	return first.Value.(*cmodel.GraphItem)
}
// 将items加入GraphItemMap，新数据再链表头部
func (this *GraphItemMap) PushAll(key string, items []*cmodel.GraphItem) error {
	this.Lock()
	defer this.Unlock()
	idx := hashKey(key) % uint32(this.Size)
	sl, ok := this.A[idx][key]
	if !ok {
		return errors.New("not exist")
	}
	sl.PushAll(items)
	return nil
}

/*
获取key对应的SafeLinkedList的状态
 */
func (this *GraphItemMap) GetFlag(key string) (uint32, error) {
	this.Lock()
	defer this.Unlock()
	idx := hashKey(key) % uint32(this.Size) // 使用crc32.ChecksumIEEE计算idx，定位slice下标
	sl, ok := this.A[idx][key]
	if !ok {
		return 0, errors.New("not exist")
	}
	return sl.Flag, nil
}

func (this *GraphItemMap) SetFlag(key string, flag uint32) error {
	this.Lock()
	defer this.Unlock()
	idx := hashKey(key) % uint32(this.Size)
	sl, ok := this.A[idx][key]
	if !ok {
		return errors.New("not exist")
	}
	sl.Flag = flag
	return nil
}

/*
以[]*cmodel.GraphItem的形式，返回key对应的SafeLinkedList所有的元素，旧的数据在前。GraphItemMap中删除
 */
func (this *GraphItemMap) PopAll(key string) []*cmodel.GraphItem {
	this.Lock()
	defer this.Unlock()
	idx := hashKey(key) % uint32(this.Size)
	sl, ok := this.A[idx][key]
	if !ok {
		return []*cmodel.GraphItem{}
	}
	return sl.PopAll()
}
// 以[]*cmodel.GraphItem的形式，返回key对应的SafeLinkedList所有的元素，旧的数据在前
func (this *GraphItemMap) FetchAll(key string) ([]*cmodel.GraphItem, uint32) {
	this.RLock()
	defer this.RUnlock()
	idx := hashKey(key) % uint32(this.Size)
	sl, ok := this.A[idx][key]
	if !ok {
		return []*cmodel.GraphItem{}, 0
	}

	return sl.FetchAll() // 以[]*cmodel.GraphItem形式返回，旧数据在前
}

func hashKey(key string) uint32 {
	if len(key) < 64 {
		var scratch [64]byte
		copy(scratch[:], key)
		return crc32.ChecksumIEEE(scratch[:len(key)])
	}
	return crc32.ChecksumIEEE([]byte(key))
}

func getWts(key string, now int64) int64 {
	interval := int64(g.CACHE_TIME)
	return now + interval - (int64(hashKey(key)) % interval)
}

func (this *GraphItemMap) PushFront(key string,
	item *cmodel.GraphItem, md5 string, cfg *g.GlobalConfig) {
	if linkedList, exists := this.Get(key); exists { // 返回key对应的SafeLinkedList
		linkedList.PushFront(item)  // 插入list头部
	} else {  // 新建SafeLinkedList，插入map
		//log.Println("new key:", key)
		safeList := &SafeLinkedList{L: list.New()}
		safeList.L.PushFront(item)

		if cfg.Migrate.Enabled && !g.IsRrdFileExist(g.RrdFileName(
			cfg.RRD.Storage, md5, item.DsType, item.Step)) {
			safeList.Flag = g.GRAPH_F_MISS // 设置rrd文件不存在的标识，说明这个key对应的rrd文件在其他graph节点上存储
		}
		this.Set(key, safeList)
	}
}

/*
以slice形式返回map this.A[idx]的key
 */
func (this *GraphItemMap) KeysByIndex(idx int) []string {
	this.RLock()
	defer this.RUnlock()

	count := len(this.A[idx])  // map的key个数
	if count == 0 {
		return []string{}
	}

	keys := make([]string, 0, count)
	for key := range this.A[idx] {
		keys = append(keys, key)
	}

	return keys
}

/*
查询最旧的item
 */
func (this *GraphItemMap) Back(key string) *cmodel.GraphItem {
	this.RLock()
	defer this.RUnlock()
	idx := hashKey(key) % uint32(this.Size)
	L, ok := this.A[idx][key]
	if !ok {
		return nil
	}

	back := L.Back()
	if back == nil {
		return nil
	}

	return back.Value.(*cmodel.GraphItem)
}

// 指定key对应的Item数量
func (this *GraphItemMap) ItemCnt(key string) int {
	this.RLock()
	defer this.RUnlock()
	idx := hashKey(key) % uint32(this.Size)
	L, ok := this.A[idx][key]
	if !ok {
		return 0
	}
	return L.Len()
}

/*
初始化GraphItems
 */
func init() {
	size := g.CACHE_TIME / g.FLUSH_DISK_STEP
	if size < 0 {
		log.Panicf("store.init, bad size %d\n", size)
	}

	GraphItems = &GraphItemMap{
		A:    make([]map[string]*SafeLinkedList, size),
		Size: size,
	}
	for i := 0; i < size; i++ {
		GraphItems.A[i] = make(map[string]*SafeLinkedList)
	}
}
