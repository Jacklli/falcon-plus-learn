package net

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"
)

func RpcClient(network, address string, timeout time.Duration) (*rpc.Client, error) {
	conn, err := net.DialTimeout(network, address, timeout)
	if err != nil {
		return nil, err
	}
	return rpc.NewClient(conn), nil
}
/*
返回一个rpc.Client
 */
func JsonRpcClient(network, address string, timeout time.Duration) (*rpc.Client, error) {
	conn, err := net.DialTimeout(network, address, timeout)
	if err != nil {
		return nil, err
	}
	return jsonrpc.NewClient(conn), err
}
