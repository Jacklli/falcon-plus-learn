package rpc

import (
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"

	"github.com/open-falcon/falcon-plus/modules/hbs/g"
)

type Hbs int
type Agent int
/*
启动JSON-RPC server
请参考：http://www.cnblogs.com/hangxin1940/p/3256995.html、https://gist.github.com/nicerobot/8954764
 */
func Start() {
	addr := g.Config().Listen

	server := rpc.NewServer()
	// server.Register(new(filter.Filter))
	server.Register(new(Agent)) // 大致原理：先将类型对应的反射信息保存到map，调用时通过反射来调用我们自己实现的处理方法，参考下方demo
	server.Register(new(Hbs))

	l, e := net.Listen("tcp", addr)
	if e != nil {
		log.Fatalln("listen error:", e)
	} else {
		log.Println("listening", addr)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("listener accept fail:", err)
			time.Sleep(time.Duration(100) * time.Millisecond)
			continue
		}
		go server.ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}

/*
Register demo:

# cat 1.go
package main

import (
        "fmt"
        "reflect"
)

type A int

func (a *A) T1() {
        fmt.Println("In T1")
}

func (a *A) T2() {
        fmt.Println("In T2")
}

func main() {
        var a = new(A)

        t := reflect.TypeOf(a)
        v := reflect.ValueOf(a)

        M := make(map[string]reflect.Value)

        sname := reflect.Indirect(v).Type().Name()
        M[sname] = v  # 注册服务信息

        M["A"].MethodByName("T1").Call([]reflect.Value{})  # 通过反射调用

        fmt.Println("====== list all ======")
        fmt.Println(t.NumMethod())
        for m := 0; m < t.NumMethod(); m++ {
                method := t.Method(m)
                fmt.Printf("%s: %v\n", method.Name, method.Type)

                v.Method(m).Call([]reflect.Value{})
        }
}

# go run 1.go
In T1
====== list all ======
2
T1: func(*main.A)
In T1
T2: func(*main.A)
In T2

 */