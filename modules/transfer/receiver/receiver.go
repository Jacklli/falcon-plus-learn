package receiver

import (
	"github.com/open-falcon/falcon-plus/modules/transfer/receiver/rpc"
	"github.com/open-falcon/falcon-plus/modules/transfer/receiver/socket"
)

func Start() {
	go rpc.StartRpc() // 启动rpc server，调用transfer.Update将agent上报的item，放入对应的发送队列
	go socket.StartSocket() // 启动tcp server，支持使用telnet方式按行上报item，调用socketTelnetHandle将上报的item，放入对应的发送队列
}
