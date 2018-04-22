package http

import (
	"github.com/gin-gonic/gin"
	"github.com/open-falcon/falcon-plus/modules/alarm/g"
	"log"
)

/*
启动http服务（使用gin框架实现），提供版本查询、健康查询、工作查询接口
 */
func Start() {
	if !g.Config().Http.Enabled {
		return
	}
	addr := g.Config().Http.Listen
	if addr == "" {
		return
	}

	r := gin.Default()
	r.GET("/version", Version)
	r.GET("/health", Health)
	r.GET("/workdir", Workdir)
	r.Run(addr)

	log.Println("http listening", addr)
}
