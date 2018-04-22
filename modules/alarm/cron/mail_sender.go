package cron

import (
	log "github.com/Sirupsen/logrus"
	"github.com/open-falcon/falcon-plus/modules/alarm/g"
	"github.com/open-falcon/falcon-plus/modules/alarm/model"
	"github.com/open-falcon/falcon-plus/modules/alarm/redi"
	"github.com/toolkits/net/httplib"
	"time"
)

func ConsumeMail() {
	for {
		L := redi.PopAllMail() // 循环读取/mail队列中的内容，直到空或出错，以[]*model.Mail的形式返回
		if len(L) == 0 {
			time.Sleep(time.Millisecond * 200)
			continue
		}
		SendMailList(L)
	}
}

func SendMailList(L []*model.Mail) {
	for _, mail := range L {
		MailWorkerChan <- 1
		go SendMail(mail)
	}
}

func SendMail(mail *model.Mail) {
	defer func() {
		<-MailWorkerChan
	}()

	url := g.Config().Api.Mail // "mail": "http://127.0.0.1:10086/mail", //邮件发送网关地址
	r := httplib.Post(url).SetTimeout(5*time.Second, 30*time.Second)
	r.Param("tos", mail.Tos)
	r.Param("subject", mail.Subject)
	r.Param("content", mail.Content)
	resp, err := r.String()
	if err != nil {
		log.Errorf("send mail fail, receiver:%s, subject:%s, cotent:%s, error:%v", mail.Tos, mail.Subject, mail.Content, err)
	}

	log.Debugf("send mail:%v, resp:%v, url:%s", mail, resp, url)
}
