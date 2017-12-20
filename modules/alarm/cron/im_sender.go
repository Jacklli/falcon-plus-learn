package cron

import (
	log "github.com/Sirupsen/logrus"
	"github.com/open-falcon/falcon-plus/modules/alarm/g"
	"github.com/open-falcon/falcon-plus/modules/alarm/model"
	"github.com/open-falcon/falcon-plus/modules/alarm/redi"
	"github.com/toolkits/net/httplib"
	"time"
)

/*
读取/im队列中的内容，调用微信发送网关地址发送
 */
func ConsumeIM() {
	for {
		L := redi.PopAllIM() // 循环读取/im队列中的内容，直到空或出错，以[]*model.IM的形式返回
		if len(L) == 0 {
			time.Sleep(time.Millisecond * 200)
			continue
		}
		SendIMList(L)
	}
}

func SendIMList(L []*model.IM) {
	for _, im := range L {
		IMWorkerChan <- 1 // 发送之前，尝试写IMWorkerChan，达到并发控制，类似信号量
		go SendIM(im) // 开启单独的goroutine执行发送动作
	}
}

/*
调用微信发送网关地址发送
 */
func SendIM(im *model.IM) {
	defer func() {
		<-IMWorkerChan
	}()

	url := g.Config().Api.IM // "im": "http://127.0.0.1:10086/wechat",  // 微信发送网关地址
	r := httplib.Post(url).SetTimeout(5*time.Second, 30*time.Second)
	r.Param("tos", im.Tos)
	r.Param("content", im.Content)
	resp, err := r.String()
	if err != nil {
		log.Errorf("send im fail, tos:%s, cotent:%s, error:%v", im.Tos, im.Content, err)
	}

	log.Debugf("send im:%v, resp:%v, url:%s", im, resp, url)
}
