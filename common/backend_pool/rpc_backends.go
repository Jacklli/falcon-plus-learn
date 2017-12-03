// Copyright 2017 Xiaomi, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package backend_pool

import (
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"sync"
	"time"

	connp "github.com/toolkits/conn_pool"
	rpcpool "github.com/toolkits/conn_pool/rpc_conn_pool"
)

// ConnPools Manager
type SafeRpcConnPools struct {
	sync.RWMutex
	M           map[string]*connp.ConnPool
	MaxConns    int
	MaxIdle     int
	ConnTimeout int
	CallTimeout int
}
/*
遍历cluster，针对每一个address，创建一个ConnPool，维护在SafeRpcConnPools.M中，返回SafeRpcConnPools地址
 */
func CreateSafeRpcConnPools(maxConns, maxIdle, connTimeout, callTimeout int, cluster []string) *SafeRpcConnPools {
	cp := &SafeRpcConnPools{M: make(map[string]*connp.ConnPool), MaxConns: maxConns, MaxIdle: maxIdle,
		ConnTimeout: connTimeout, CallTimeout: callTimeout}

	ct := time.Duration(cp.ConnTimeout) * time.Millisecond
	for _, address := range cluster {
		if _, exist := cp.M[address]; exist {
			continue
		}
		cp.M[address] = createOneRpcPool(address, address, ct, maxConns, maxIdle) // 根据(name, address)创建一个ConnPool，返回其地址
	}

	return cp
}

func CreateSafeJsonrpcConnPools(maxConns, maxIdle, connTimeout, callTimeout int, cluster []string) *SafeRpcConnPools {
	cp := &SafeRpcConnPools{M: make(map[string]*connp.ConnPool), MaxConns: maxConns, MaxIdle: maxIdle,
		ConnTimeout: connTimeout, CallTimeout: callTimeout}

	ct := time.Duration(cp.ConnTimeout) * time.Millisecond
	for _, address := range cluster {
		if _, exist := cp.M[address]; exist {
			continue
		}
		cp.M[address] = createOneJsonrpcPool(address, address, ct, maxConns, maxIdle)
	}

	return cp
}
/*
从ConnPool中获取一个连接进行发送
 */
// 同步发送, 完成发送或超时后 才能返回
func (this *SafeRpcConnPools) Call(addr, method string, args interface{}, resp interface{}) error {
	connPool, exists := this.Get(addr) // 返回addr对应的ConnPool
	if !exists {
		return fmt.Errorf("%s has no connection pool", addr)
	}

	conn, err := connPool.Fetch() // 从ConnPool获取一个可用连接，连接数不足的话动态创建
	if err != nil {
		return fmt.Errorf("%s get connection fail: conn %v, err %v. proc: %s", addr, conn, err, connPool.Proc())
	}

	rpcClient := conn.(*rpcpool.RpcClient)
	callTimeout := time.Duration(this.CallTimeout) * time.Millisecond

	done := make(chan error, 1)
	go func() { // 开启goroutine执行rpc调用
		done <- rpcClient.Call(method, args, resp)
	}()

	select {
	case <-time.After(callTimeout): // 超时处理
		connPool.ForceClose(conn) // 从ConnPool中删除超时连接
		return fmt.Errorf("%s, call timeout", addr)
	case err = <-done:
		if err != nil {
			connPool.ForceClose(conn) // 从ConnPool中删除异常连接
			err = fmt.Errorf("%s, call failed, err %v. proc: %s", addr, err, connPool.Proc())
		} else {
			connPool.Release(conn) // 归还连接到connPool
		}
		return err
	}
}
/*
返回address对应的ConnPool
 */
func (this *SafeRpcConnPools) Get(address string) (*connp.ConnPool, bool) {
	this.RLock()
	defer this.RUnlock()
	p, exists := this.M[address]
	return p, exists
}

func (this *SafeRpcConnPools) Destroy() {
	this.Lock()
	defer this.Unlock()
	addresses := make([]string, 0, len(this.M))
	for address := range this.M {
		addresses = append(addresses, address)
	}

	for _, address := range addresses {
		this.M[address].Destroy()
		delete(this.M, address)
	}
}
/*
返回ConnPool的统计信息
 */
func (this *SafeRpcConnPools) Proc() []string {
	procs := []string{}
	for _, cp := range this.M {
		procs = append(procs, cp.Proc())
	}
	return procs
}
/*
根据(name, address)创建一个ConnPool，返回其地址
需要指明一个连接创建函数，当ConnPool连接不足时调用。该函数的返回值需要满足接口NConn，即具有Close(),Name(),Closed()方法
 */
func createOneRpcPool(name string, address string, connTimeout time.Duration, maxConns int, maxIdle int) *connp.ConnPool {
	p := connp.NewConnPool(name, address, int32(maxConns), int32(maxIdle)) // 创建一个ConnPool，返回其地址
	p.New = func(connName string) (connp.NConn, error) { // 连接创建函数，当一个ConnPool连接不足时，调用
		_, err := net.ResolveTCPAddr("tcp", p.Address)
		if err != nil {
			return nil, err
		}

		conn, err := net.DialTimeout("tcp", p.Address, connTimeout)
		if err != nil {
			return nil, err
		}

		/*
		返回一个&RpcClient{cli: rpc.NewClient(conn), name: connName}
		RpcClient满足接口NConn，具有Close(),Name(),Closed()方法
		 */
		return rpcpool.NewRpcClient(rpc.NewClient(conn), connName), nil
	}

	return p
}

func createOneJsonrpcPool(name string, address string, connTimeout time.Duration, maxConns int, maxIdle int) *connp.ConnPool {
	p := connp.NewConnPool(name, address, int32(maxConns), int32(maxIdle))
	p.New = func(connName string) (connp.NConn, error) {
		_, err := net.ResolveTCPAddr("tcp", p.Address)
		if err != nil {
			return nil, err
		}

		conn, err := net.DialTimeout("tcp", p.Address, connTimeout)
		if err != nil {
			return nil, err
		}

		return rpcpool.NewRpcClientWithCodec(jsonrpc.NewClientCodec(conn), connName), nil
	}

	return p
}
