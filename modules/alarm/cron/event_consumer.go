package cron

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"

	cmodel "github.com/open-falcon/falcon-plus/common/model"
	"github.com/open-falcon/falcon-plus/modules/alarm/api"
	"github.com/open-falcon/falcon-plus/modules/alarm/g"
	"github.com/open-falcon/falcon-plus/modules/alarm/redi"
)

/*
调用callback，并发送短信、IM、邮件到redis的待发送队列（高优先级报警不合并）或待合并队列（低优先级报警）
 */
func consume(event *cmodel.Event, isHigh bool) {
	actionId := event.ActionId()
	if actionId <= 0 {
		return
	}

	action := api.GetAction(actionId) // 获取actionid对应的action
	if action == nil {
		return
	}

	if action.Callback == 1 {
		HandleCallback(event, action) // 发送短信、IM、邮件到对应的redis队列，并调用外部接口执行回调函数
	}

	if isHigh {
		consumeHighEvents(event, action) // 高优先级的报警不合并，直接发送到redis的待发送队列
	} else {
		consumeLowEvents(event, action) // 低优先级的报警发送到redis的待合并队列
	}
}

// 高优先级的不做报警合并
func consumeHighEvents(event *cmodel.Event, action *api.Action) {
	if action.Uic == "" {
		return
	}

	phones, mails, ims := api.ParseTeams(action.Uic) // 返回phoneSet,mailSet,imSet

	// 生成待发送内容
	smsContent := GenerateSmsContent(event)
	mailContent := GenerateMailContent(event)
	imContent := GenerateIMContent(event)

	// 将内容写入redis对应队列
	// <=P2 才发送短信
	if event.Priority() < 3 {
		redi.WriteSms(phones, smsContent)
	}

	redi.WriteIM(ims, imContent)
	redi.WriteMail(mails, smsContent, mailContent)

}

// 低优先级的做报警合并
func consumeLowEvents(event *cmodel.Event, action *api.Action) {
	if action.Uic == "" {
		return
	}

	// <=P2 才发送短信
	if event.Priority() < 3 {
		ParseUserSms(event, action) // 将报警短信信息LPUSH到redis队列/queue/user/sms
	}

	ParseUserIm(event, action) // 将报警IM信息LPUSH到redis队列/queue/user/im
	ParseUserMail(event, action) // 将报警邮件信息LPUSH到redis队列/queue/user/mail
}

func ParseUserSms(event *cmodel.Event, action *api.Action) {
	userMap := api.GetUsers(action.Uic) // 查询team对应的成员列表，以map的形式返回

	content := GenerateSmsContent(event) // 生成短信内容
	metric := event.Metric()
	status := event.Status
	priority := event.Priority()

	queue := g.Config().Redis.UserSmsQueue // /queue/user/sms

	rc := g.RedisConnPool.Get()
	defer rc.Close()

	for _, user := range userMap {
		dto := SmsDto{
			Priority: priority,
			Metric:   metric,
			Content:  content,
			Phone:    user.Phone,
			Status:   status,
		}
		bs, err := json.Marshal(dto)
		if err != nil {
			log.Error("json marshal SmsDto fail:", err)
			continue
		}

		// 将报警短信信息LPUSH到redis队列/queue/user/sms
		_, err = rc.Do("LPUSH", queue, string(bs))
		if err != nil {
			log.Error("LPUSH redis", queue, "fail:", err, "dto:", string(bs))
		}
	}
}

func ParseUserMail(event *cmodel.Event, action *api.Action) {
	userMap := api.GetUsers(action.Uic)

	metric := event.Metric()
	subject := GenerateSmsContent(event)
	content := GenerateMailContent(event)
	status := event.Status
	priority := event.Priority()

	queue := g.Config().Redis.UserMailQueue

	rc := g.RedisConnPool.Get()
	defer rc.Close()

	for _, user := range userMap {
		dto := MailDto{
			Priority: priority,
			Metric:   metric,
			Subject:  subject,
			Content:  content,
			Email:    user.Email,
			Status:   status,
		}
		bs, err := json.Marshal(dto)
		if err != nil {
			log.Error("json marshal MailDto fail:", err)
			continue
		}

		_, err = rc.Do("LPUSH", queue, string(bs))
		if err != nil {
			log.Error("LPUSH redis", queue, "fail:", err, "dto:", string(bs))
		}
	}
}

func ParseUserIm(event *cmodel.Event, action *api.Action) {
	userMap := api.GetUsers(action.Uic)

	content := GenerateIMContent(event)
	metric := event.Metric()
	status := event.Status
	priority := event.Priority()

	queue := g.Config().Redis.UserIMQueue

	rc := g.RedisConnPool.Get()
	defer rc.Close()

	for _, user := range userMap {
		dto := ImDto{
			Priority: priority,
			Metric:   metric,
			Content:  content,
			IM:       user.IM,
			Status:   status,
		}
		bs, err := json.Marshal(dto)
		if err != nil {
			log.Error("json marshal ImDto fail:", err)
			continue
		}

		_, err = rc.Do("LPUSH", queue, string(bs))
		if err != nil {
			log.Error("LPUSH redis", queue, "fail:", err, "dto:", string(bs))
		}
	}
}
