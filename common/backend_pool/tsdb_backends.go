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
	"time"

	connp "github.com/toolkits/conn_pool"
)

// TSDB
type TsdbClient struct {
	cli  net.Conn
	name string
}

func (t TsdbClient) Name() string {
	return t.name
}

func (t TsdbClient) Closed() bool {
	return t.cli == nil
}

func (t TsdbClient) Close() error {
	if t.cli != nil {
		err := t.cli.Close()
		t.cli = nil
		return err
	}
	return nil
}
/*
根据address创建一个ConnPool，返回其地址
需要指明一个连接创建函数，当ConnPool连接不足时调用。该函数的返回值需要满足接口NConn，即具有Close(),Name(),Closed()方法
 */
func newTsdbConnPool(address string, maxConns int, maxIdle int, connTimeout int) *connp.ConnPool {
	pool := connp.NewConnPool("tsdb", address, int32(maxConns), int32(maxIdle)) // 创建一个ConnPool，返回其地址

	pool.New = func(name string) (connp.NConn, error) { // 连接创建函数，当一个ConnPool连接不足时，调用
		_, err := net.ResolveTCPAddr("tcp", address)
		if err != nil {
			return nil, err
		}

		conn, err := net.DialTimeout("tcp", address, time.Duration(connTimeout)*time.Millisecond)
		if err != nil {
			return nil, err
		}
		/*
		返回一个TsdbClient{cli: conn, name: name}
		TsdbClient满足接口NConn，具有Close(),Name(),Closed()方法
		 */
		return TsdbClient{conn, name}, nil
	}

	return pool
}

type TsdbConnPoolHelper struct {
	p           *connp.ConnPool
	maxConns    int
	maxIdle     int
	connTimeout int
	callTimeout int
	address     string
}
/*
创建一个TsdbConnPoolHelper，返回其地址
 */
func NewTsdbConnPoolHelper(address string, maxConns, maxIdle, connTimeout, callTimeout int) *TsdbConnPoolHelper {
	return &TsdbConnPoolHelper{
		p:           newTsdbConnPool(address, maxConns, maxIdle, connTimeout), // 根据address创建一个ConnPool，返回其地址
		maxConns:    maxConns,
		maxIdle:     maxIdle,
		connTimeout: connTimeout,
		callTimeout: callTimeout,
		address:     address,
	}
}
/*
从TsdbConnPoolHelper.p获取一个可用连进行发送
 */
func (t *TsdbConnPoolHelper) Send(data []byte) (err error) {
	conn, err := t.p.Fetch() // 从ConnPool获取一个可用连接，连接数不足的话动态创建
	if err != nil {
		return fmt.Errorf("get connection fail: err %v. proc: %s", err, t.p.Proc())
	}

	cli := conn.(TsdbClient).cli // 使用type assertion，将conn转换成TsdbClient

	done := make(chan error, 1)
	go func() {
		_, err = cli.Write(data) // 发送数据
		done <- err
	}()

	select {
	case <-time.After(time.Duration(t.callTimeout) * time.Millisecond): // 超时处理
		t.p.ForceClose(conn) // 从ConnPool中删除超时连接
		return fmt.Errorf("%s, call timeout", t.address)
	case err = <-done:
		if err != nil {
			t.p.ForceClose(conn) // 从ConnPool中删除异常连接
			err = fmt.Errorf("%s, call failed, err %v. proc: %s", t.address, err, t.p.Proc())
		} else {
			t.p.Release(conn) // 归还连接到ConnPool
		}
		return err
	}
}

func (t *TsdbConnPoolHelper) Destroy() {
	if t.p != nil {
		t.p.Destroy()
	}
}
