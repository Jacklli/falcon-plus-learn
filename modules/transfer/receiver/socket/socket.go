package socket

import (
	"github.com/open-falcon/falcon-plus/modules/transfer/g"
	"log"
	"net"
)
/*
支持使用telnet方式按行上报item
 */
func StartSocket() {
	if !g.Config().Socket.Enabled {
		return
	}

	addr := g.Config().Socket.Listen
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		log.Fatalf("net.ResolveTCPAddr fail: %s", err)
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Fatalf("listen %s fail: %s", addr, err)
	} else {
		log.Println("socket listening", addr)
	}

	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("listener.Accept occur error:", err)
			continue
		}

		go socketTelnetHandle(conn) // 使用telnet，按行读取item，转换成*cmodel.MetaData放入发送队列
	}
}
