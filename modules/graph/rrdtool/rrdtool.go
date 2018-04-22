package rrdtool

import (
	"errors"
	"log"
	"math"
	"sync/atomic"
	"time"

	cmodel "github.com/open-falcon/falcon-plus/common/model"
	"github.com/open-falcon/rrdlite"
	"github.com/toolkits/file"

	"github.com/open-falcon/falcon-plus/modules/graph/g"
	"github.com/open-falcon/falcon-plus/modules/graph/store"
)

var (
	disk_counter uint64
	net_counter  uint64
)

type fetch_t struct {
	filename string
	cf       string
	start    int64
	end      int64
	step     int
	data     []*cmodel.RRDData
}

type flushfile_t struct {
	filename string
	items    []*cmodel.GraphItem
}

type readfile_t struct {
	filename string
	data     []byte
}
// 开启net_task_worker goroutine监听net_task任务管道用于同其他graph传输监控数据
// 开启ioworker goroutine监听io_task任务管道用于读写本地rrd文件
// 开启syncdisk goroutine用于周期性刷新GraphItems缓存到rrd文件
func Start() {
	cfg := g.Config() // 返回配置信息
	var err error
	// check data dir
	/*
    "rrd": {
        "storage": "./data/6070" // 历史数据的文件存储路径（如有必要，请修改为合适的路）
    },
	 */
	if err = file.EnsureDirRW(cfg.RRD.Storage); err != nil {  // 确保存储路径存在且可读写
		log.Fatalln("rrdtool.Start error, bad data dir "+cfg.RRD.Storage+",", err)
	}

	/*
	将扩容前的graph节点加入一致性hash，并针对每个graph节点创建任务管道Net_task_ch[node]和rpc连接clients[node]
	开启cfg.Migrate.Concurrency个net_task_worker，用于和扩容前的graph传输监控数据
	*/
	migrate_start(cfg)

	// sync disk
	go syncDisk()  // 使用异步方式周期性刷新GraphItems缓存到文件
	go ioWorker()  // 从io_task_chan读取task进行处理，如flushrrd
	log.Println("rrdtool.Start ok")
}

// RRA.Point.Size
const (
	RRA1PointCnt   = 720 // 1m一个点存12h
	RRA5PointCnt   = 576 // 5m一个点存2d
	RRA20PointCnt  = 504 // 20m一个点存7d
	RRA180PointCnt = 766 // 3h一个点存3month
	RRA720PointCnt = 730 // 12h一个点存1year
)

/*
创建rrd文件，设置rrd属性
 */
func create(filename string, item *cmodel.GraphItem) error {
	now := time.Now()
	start := now.Add(time.Duration(-24) * time.Hour)
	step := uint(item.Step)

	c := rrdlite.NewCreator(filename, start, step)
	c.DS("metric", item.DsType, item.Heartbeat, item.Min, item.Max)

	// 设置各种归档策略
	// 1分钟一个点存 12小时
	c.RRA("AVERAGE", 0, 1, RRA1PointCnt)

	// 5m一个点存2d
	c.RRA("AVERAGE", 0, 5, RRA5PointCnt)
	c.RRA("MAX", 0, 5, RRA5PointCnt)
	c.RRA("MIN", 0, 5, RRA5PointCnt)

	// 20m一个点存7d
	c.RRA("AVERAGE", 0, 20, RRA20PointCnt)
	c.RRA("MAX", 0, 20, RRA20PointCnt)
	c.RRA("MIN", 0, 20, RRA20PointCnt)

	// 3小时一个点存3个月
	c.RRA("AVERAGE", 0, 180, RRA180PointCnt)
	c.RRA("MAX", 0, 180, RRA180PointCnt)
	c.RRA("MIN", 0, 180, RRA180PointCnt)

	// 12小时一个点存1year
	c.RRA("AVERAGE", 0, 720, RRA720PointCnt)
	c.RRA("MAX", 0, 720, RRA720PointCnt)
	c.RRA("MIN", 0, 720, RRA720PointCnt)

	return c.Create(true)
}

func update(filename string, items []*cmodel.GraphItem) error {
	u := rrdlite.NewUpdater(filename)

	for _, item := range items {
		v := math.Abs(item.Value)
		if v > 1e+300 || (v < 1e-300 && v > 0) {
			continue
		}
		if item.DsType == "DERIVE" || item.DsType == "COUNTER" {
			u.Cache(item.Timestamp, int(item.Value))
		} else {
			u.Cache(item.Timestamp, item.Value)
		}
	}

	return u.Update()
}

// flush to disk from memory
// 最新的数据在列表的最后面
// TODO fix me, filename fmt from item[0], it's hard to keep consistent
func flushrrd(filename string, items []*cmodel.GraphItem) error {
	if items == nil || len(items) == 0 {
		return errors.New("empty items")
	}

	if !g.IsRrdFileExist(filename) {
		baseDir := file.Dir(filename)

		err := file.InsureDir(baseDir)
		if err != nil {
			return err
		}

		err = create(filename, items[0]) // 创建rrd文件，设置rrd属性
		if err != nil {
			return err
		}
	}

	return update(filename, items) // 更新rrd文件
}

/*
发送IO_TASK_M_READ(包括参数)指令到管道io_task_chan，供ioWorker处理
并通过done等待操作完成
返回rrd文件内容
 */
func ReadFile(filename string) ([]byte, error) {
	done := make(chan error, 1)
	task := &io_task_t{
		method: IO_TASK_M_READ,
		args:   &readfile_t{filename: filename},
		done:   done,
	}

	io_task_chan <- task
	err := <-done
	return task.args.(*readfile_t).data, err
}

/*
发送IO_TASK_M_FLUSH(包括参数)指令到管道io_task_chan，供ioWorker处理
并通过done等待操作完成
刷新rrd文件
 */
func FlushFile(filename string, items []*cmodel.GraphItem) error {
	done := make(chan error, 1)  // 接收处理结果的channel
	io_task_chan <- &io_task_t{
		method: IO_TASK_M_FLUSH,
		args: &flushfile_t{
			filename: filename,
			items:    items,
		},
		done: done,
	}
	atomic.AddUint64(&disk_counter, 1)  // 统计信息
	return <-done
}

func Fetch(filename string, cf string, start, end int64, step int) ([]*cmodel.RRDData, error) {
	done := make(chan error, 1)
	task := &io_task_t{
		method: IO_TASK_M_FETCH,
		args: &fetch_t{
			filename: filename,
			cf:       cf,
			start:    start,
			end:      end,
			step:     step,
		},
		done: done,
	}
	io_task_chan <- task
	err := <-done
	return task.args.(*fetch_t).data, err
}
// 查询rrd文件
func fetch(filename string, cf string, start, end int64, step int) ([]*cmodel.RRDData, error) {
	start_t := time.Unix(start, 0)
	end_t := time.Unix(end, 0)
	step_t := time.Duration(step) * time.Second

	fetchRes, err := rrdlite.Fetch(filename, cf, start_t, end_t, step_t)
	if err != nil {
		return []*cmodel.RRDData{}, err
	}

	defer fetchRes.FreeValues()

	values := fetchRes.Values()
	size := len(values)
	ret := make([]*cmodel.RRDData, size)

	start_ts := fetchRes.Start.Unix()
	step_s := fetchRes.Step.Seconds()

	for i, val := range values {
		ts := start_ts + int64(i+1)*int64(step_s)
		d := &cmodel.RRDData{
			Timestamp: ts,
			Value:     cmodel.JsonFloat(val),
		}
		ret[i] = d
	}

	return ret, nil
}

func FlushAll(force bool) {
	n := store.GraphItems.Size / 10
	for i := 0; i < store.GraphItems.Size; i++ {
		FlushRRD(i, force)
		if i%n == 0 {
			log.Printf("flush hash idx:%03d size:%03d disk:%08d net:%08d\n",
				i, store.GraphItems.Size, disk_counter, net_counter)
		}
	}
	log.Printf("flush hash done (disk:%08d net:%08d)\n", disk_counter, net_counter)
}

/*
将key对应的GraphItems flush到rrd文件
 */
func CommitByKey(key string) {

	md5, dsType, step, err := g.SplitRrdCacheKey(key) // split key提取md5, dsType, step
	if err != nil {
		return
	}
	filename := g.RrdFileName(g.Config().RRD.Storage, md5, dsType, step) // 使用md5, dsType, step构造rrd文件名

	items := store.GraphItems.PopAll(key) // 以[]*cmodel.GraphItem的形式，返回key对应的SafeLinkedList所有的元素，旧的数据在前
	if len(items) == 0 {
		return
	}
	FlushFile(filename, items) // 发送IO_TASK_M_FLUSH(包括参数)指令到管道io_task_chan，供ioWorker处理
}

func PullByKey(key string) {
	done := make(chan error)

	item := store.GraphItems.First(key)
	if item == nil {
		return
	}
	node, err := Consistent.Get(item.PrimaryKey())
	if err != nil {
		return
	}
	Net_task_ch[node] <- &Net_task_t{
		Method: NET_TASK_M_PULL,
		Key:    key,
		Done:   done,
	}
	// net_task slow, shouldn't block syncDisk() or FlushAll()
	// warning: recev sigout when migrating, maybe lost memory data
	go func() {
		err := <-done
		if err != nil {
			log.Printf("get %s from remote err[%s]\n", key, err)
			return
		}
		atomic.AddUint64(&net_counter, 1)
		//todo: flushfile after getfile? not yet
	}()
}

/*
将下标为idx的map中的GraphItem刷新到文件
 */
func FlushRRD(idx int, force bool) {
	begin := time.Now()
	atomic.StoreInt32(&flushrrd_timeout, 0)

	keys := store.GraphItems.KeysByIndex(idx) // 以slice形式返回map store.GraphItems.A[idx]的key
	if len(keys) == 0 {
		return
	}

	for _, key := range keys {
		flag, _ := store.GraphItems.GetFlag(key) // 获取key对应的SafeLinkedList的状态

		//write err data to local filename
		if force == false && g.Config().Migrate.Enabled && flag&g.GRAPH_F_MISS != 0 { // 从扩容前的graph节点将rrd文件拷贝到本地
			if time.Since(begin) > time.Millisecond*g.FLUSH_DISK_STEP {
				atomic.StoreInt32(&flushrrd_timeout, 1)
			}
			PullByKey(key)
		} else if force || shouldFlush(key) {  // 判断是否达到flush阈值，1、根据数量；2、根据时间间隔
			CommitByKey(key) // 将key对应的GraphItems flush到rrd文件
		}
	}
}

/*
判断是否达到flush阈值，1、根据数量；2、根据时间间隔
 */
func shouldFlush(key string) bool {

	if store.GraphItems.ItemCnt(key) >= g.FLUSH_MIN_COUNT { // 计算key对应的SafeLinkedList的长度是否>=FLUSH_MIN_COUNT
		return true
	}

	deadline := time.Now().Unix() - int64(g.FLUSH_MAX_WAIT)
	back := store.GraphItems.Back(key)  // 查询最旧的item
	if back != nil && back.Timestamp <= deadline { // 计算时长是否超过FLUSH_MAX_WAIT没有flush过
		return true
	}

	return false
}
