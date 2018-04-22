package cron

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"strings"
	"time"

	"github.com/open-falcon/falcon-plus/common/model"
	"github.com/open-falcon/falcon-plus/modules/alarm/api"
	"github.com/open-falcon/falcon-plus/modules/alarm/redi"
	"github.com/toolkits/net/httplib"
)

/*
发送短信、IM、邮件到对应的redis队列，并调用外部接口执行回调函数
 */
func HandleCallback(event *model.Event, action *api.Action) {

	teams := action.Uic
	phones := []string{}
	mails := []string{}
	ims := []string{}

	if teams != "" {
		phones, mails, ims = api.ParseTeams(teams) // 根据teams查询到成员列表，将所有成员的phone、mail、im分别将入set，并返回phoneSet,mailSet,imSet
		smsContent := GenerateSmsContent(event) // 根据event生成短信内容
		mailContent := GenerateMailContent(event) // 根据event生成邮件内容
		imContent := GenerateIMContent(event) // 根据event生成IM内容
		if action.BeforeCallbackSms == 1 {
			redi.WriteSms(phones, smsContent) // 放到redis的/sms队列
			redi.WriteIM(ims, imContent) // 放到redis的/im队列
		}

		if action.BeforeCallbackMail == 1 {
			redi.WriteMail(mails, smsContent, mailContent) // 放到redis的/mail队列
		}
	}

	// 通过GET action.Url，调用外部接口执行回调函数
	message := Callback(event, action)

	if teams != "" {
		if action.AfterCallbackSms == 1 {
			redi.WriteSms(phones, message)
			redi.WriteIM(ims, message)
		}

		if action.AfterCallbackMail == 1 {
			redi.WriteMail(mails, message, message)
		}
	}

}

/*
通过GET action.Url，调用外部接口执行回调函数
 */
func Callback(event *model.Event, action *api.Action) string {
	if action.Url == "" {
		return "callback url is blank"
	}

	// 构造tags字符串：k1:v1,k2:v2,k3:v3
	L := make([]string, 0)
	if len(event.PushedTags) > 0 {
		for k, v := range event.PushedTags {
			L = append(L, fmt.Sprintf("%s:%s", k, v))
		}
	}

	tags := ""
	if len(L) > 0 {
		tags = strings.Join(L, ",")
	}

	req := httplib.Get(action.Url).SetTimeout(3*time.Second, 20*time.Second)

	req.Param("endpoint", event.Endpoint)
	req.Param("metric", event.Metric())
	req.Param("status", event.Status)
	req.Param("step", fmt.Sprintf("%d", event.CurrentStep))
	req.Param("priority", fmt.Sprintf("%d", event.Priority()))
	req.Param("time", event.FormattedTime())
	req.Param("tpl_id", fmt.Sprintf("%d", event.TplId()))
	req.Param("exp_id", fmt.Sprintf("%d", event.ExpressionId()))
	req.Param("stra_id", fmt.Sprintf("%d", event.StrategyId()))
	req.Param("tags", tags)

	resp, e := req.String()

	success := "success"
	if e != nil {
		log.Errorf("callback fail, action:%v, event:%s, error:%s", action, event.String(), e.Error())
		success = fmt.Sprintf("fail:%s", e.Error())
	}
	message := fmt.Sprintf("curl %s %s. resp: %s", action.Url, success, resp)
	log.Debugf("callback to url:%s, event:%s, resp:%s", action.Url, event.String(), resp)

	return message
}
