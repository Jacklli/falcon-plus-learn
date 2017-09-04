package conn_pool

import (
	"fmt"
	"io"
	"sync"
	"time"
)

var ErrMaxConn = fmt.Errorf("maximum connections reached")

//named conn
type NConn interface {
	io.Closer
	Name() string
	Closed() bool
}

//conn_pool
type ConnPool struct {
	sync.RWMutex

	Name     string
	Address  string
	MaxConns int32
	MaxIdle  int32
	Cnt      int64  // 总连接数

	New func(name string) (NConn, error) // 连接创建函数，在pool中连接不足时调用

	active int32 // 使用中的连接数
	free   []NConn // 空闲连接
	all    map[string]NConn // 所有连接信息
}
/*
创建一个ConnPool，返回其地址

//conn_pool
type ConnPool struct {
	sync.RWMutex

	Name     string
	Address  string
	MaxConns int32
	MaxIdle  int32
	Cnt      int64 // 总连接数

	New func(name string) (NConn, error) // 连接创建函数，在pool中连接不足时调用

	active int32 // 使用中的连接数
	free   []NConn // 空闲连接
	all    map[string]NConn // 所有连接信息
}

//named conn
type NConn interface {
	io.Closer
	Name() string
	Closed() bool
}
 */
func NewConnPool(name string, address string, maxConns int32, maxIdle int32) *ConnPool {
	return &ConnPool{Name: name, Address: address, MaxConns: maxConns, MaxIdle: maxIdle, Cnt: 0, all: make(map[string]NConn)}
}

func (this *ConnPool) Proc() string {
	this.RLock()
	defer this.RUnlock()

	return fmt.Sprintf("Name:%s,Cnt:%d,active:%d,all:%d,free:%d",
		this.Name, this.Cnt, this.active, len(this.all), len(this.free))
}
/*
从ConnPool获取一个可用连接，连接数不足的话动态创建
 */
func (this *ConnPool) Fetch() (NConn, error) {
	this.Lock()
	defer this.Unlock()

	// get from free
	conn := this.fetchFree() // 从ConnPool中获取一个可用连接
	if conn != nil {
		return conn, nil
	}

	if this.overMax() { // 判断连接数是否已经达到最大值
		return nil, ErrMaxConn
	}

	// create new conn
	conn, err := this.newConn() // 调用New，创建新的连接
	if err != nil {
		return nil, err
	}

	this.increActive() // 增加active计数
	return conn, nil
}
/*

 */
func (this *ConnPool) Release(conn NConn) {
	this.Lock()
	defer this.Unlock()

	if this.overMaxIdle() { // 连接数超限，直接删除连接
		this.deleteConn(conn)
		this.decreActive()
	} else { // 将连接添加到free列表
		this.addFree(conn)
	}
}
/*
从ConnPool中删除连接
 */
func (this *ConnPool) ForceClose(conn NConn) {
	this.Lock()
	defer this.Unlock()

	this.deleteConn(conn) // 从all中删除连接conn
	this.decreActive() // 减少active计数
}

func (this *ConnPool) Destroy() {
	this.Lock()
	defer this.Unlock()

	for _, conn := range this.free {
		if conn != nil && !conn.Closed() {
			conn.Close()
		}
	}

	for _, conn := range this.all {
		if conn != nil && !conn.Closed() {
			conn.Close()
		}
	}

	this.active = 0
	this.free = []NConn{}
	this.all = map[string]NConn{}
}
/*
调用New，创建新的连接
 */
// internal, concurrently unsafe
func (this *ConnPool) newConn() (NConn, error) {
	name := fmt.Sprintf("%s_%d_%d", this.Name, this.Cnt, time.Now().Unix())
	conn, err := this.New(name) // 调用New，创建新的连接
	if err != nil {
		if conn != nil {
			conn.Close()
		}
		return nil, err
	}

	this.Cnt++ // 增加连接序号
	this.all[conn.Name()] = conn // 保存连接
	return conn, nil
}

func (this *ConnPool) deleteConn(conn NConn) {
	if conn != nil {
		conn.Close()
	}
	delete(this.all, conn.Name())
}

func (this *ConnPool) addFree(conn NConn) {
	this.free = append(this.free, conn)
}
/*
从ConnPool中获取一个可用连接
 */
func (this *ConnPool) fetchFree() NConn {
	if len(this.free) == 0 {
		return nil
	}

	conn := this.free[0]
	this.free = this.free[1:]
	return conn
}

func (this *ConnPool) increActive() {
	this.active += 1
}

func (this *ConnPool) decreActive() {
	this.active -= 1
}
/*
判断连接数是否已经达到最大值
 */
func (this *ConnPool) overMax() bool {
	return this.active >= this.MaxConns
}

func (this *ConnPool) overMaxIdle() bool {
	return int32(len(this.free)) >= this.MaxIdle
}
