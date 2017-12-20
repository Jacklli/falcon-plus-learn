package cron

import (
	log "github.com/Sirupsen/logrus"
	"github.com/open-falcon/falcon-plus/modules/alarm/g"
	"github.com/open-falcon/falcon-plus/modules/alarm/model"
	"github.com/open-falcon/falcon-plus/modules/alarm/redi"
	"github.com/toolkits/net/httplib"
	"time"
)

func ConsumeSms() {
	for {
		L := redi.PopAllSms() // 循环读取/sms队列中的内容，直到空或出错，以[]*model.Sms的形式返回
		if len(L) == 0 {
			time.Sleep(time.Millisecond * 200)
			continue
		}
		SendSmsList(L)
	}
}

func SendSmsList(L []*model.Sms) {
	for _, sms := range L {
		SmsWorkerChan <- 1
		go SendSms(sms)
	}
}

func SendSms(sms *model.Sms) {
	defer func() {
		<-SmsWorkerChan
	}()

	url := g.Config().Api.Sms // "sms": "http://127.0.0.1:10086/sms",  //短信发送网关地址
	r := httplib.Post(url).SetTimeout(5*time.Second, 30*time.Second)
	r.Param("tos", sms.Tos)
	r.Param("content", sms.Content)
	resp, err := r.String()
	if err != nil {
		log.Errorf("send sms fail, tos:%s, cotent:%s, error:%v", sms.Tos, sms.Content, err)
	}

	log.Debugf("send sms:%v, resp:%v, url:%s", sms, resp, url)
}
