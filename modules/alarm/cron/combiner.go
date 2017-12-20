package cron

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
	"github.com/open-falcon/falcon-plus/modules/alarm/api"
	"github.com/open-falcon/falcon-plus/modules/alarm/g"
	"github.com/open-falcon/falcon-plus/modules/alarm/redi"
	"strings"
	"time"
)

func CombineSms() {
	for {
		// 每分钟读取处理一次
		time.Sleep(time.Minute)
		combineSms() // 读取/queue/user/sms队列中的短信内容，聚合后入库，并发送聚合短信
	}
}

func CombineMail() {
	for {
		// 每分钟读取处理一次
		time.Sleep(time.Minute)
		combineMail()
	}
}

func CombineIM() {
	for {
		// 每分钟读取处理一次
		time.Sleep(time.Minute)
		combineIM()
	}
}

/*
读取/queue/user/mail队列中的邮件内容，聚合发送
 */
func combineMail() {
	dtos := popAllMailDto() // 循环读取/queue/user/mail队列中的邮件内容，直到空或出错，以[]*MailDto的形式返回
	count := len(dtos)
	if count == 0 {
		return
	}

	// 邮件聚合，（Priority,Status,Email,Metric）相同的放到一起
	dtoMap := make(map[string][]*MailDto)
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("%d%s%s%s", dtos[i].Priority, dtos[i].Status, dtos[i].Email, dtos[i].Metric)
		if _, ok := dtoMap[key]; ok {
			dtoMap[key] = append(dtoMap[key], dtos[i])
		} else {
			dtoMap[key] = []*MailDto{dtos[i]}
		}
	}

	// 不要在这处理，继续写回redis，否则重启alarm很容易丢数据
	for _, arr := range dtoMap {
		size := len(arr)
		if size == 1 {
			redi.WriteMail([]string{arr[0].Email}, arr[0].Subject, arr[0].Content)
			continue
		}

		// 构造聚合的主题和正文
		subject := fmt.Sprintf("[P%d][%s] %d %s", arr[0].Priority, arr[0].Status, size, arr[0].Metric)
		contentArr := make([]string, size)
		for i := 0; i < size; i++ {
			contentArr[i] = arr[i].Content
		}
		content := strings.Join(contentArr, "\r\n")

		log.Debugf("combined mail subject:%s, content:%s", subject, content)
		redi.WriteMail([]string{arr[0].Email}, subject, content)
	}
}

/*
读取/queue/user/im队列中的IM内容，聚合后入库，并发送聚合消息
 */
func combineIM() {
	dtos := popAllImDto() // 循环读取/queue/user/im队列中的短信内容，直到空或出错，以[]*ImDto的形式返回
	count := len(dtos)
	if count == 0 {
		return
	}

	// 聚合消息，（Priority,Status,IM,Metric）相同的放到一起
	dtoMap := make(map[string][]*ImDto)
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("%d%s%s%s", dtos[i].Priority, dtos[i].Status, dtos[i].IM, dtos[i].Metric)
		if _, ok := dtoMap[key]; ok {
			dtoMap[key] = append(dtoMap[key], dtos[i])
		} else {
			dtoMap[key] = []*ImDto{dtos[i]}
		}
	}

	for _, arr := range dtoMap {
		size := len(arr)
		if size == 1 {
			redi.WriteIM([]string{arr[0].IM}, arr[0].Content)
			continue
		}

		// 把多个im内容写入数据库，只给用户提供一个链接
		contentArr := make([]string, size)
		for i := 0; i < size; i++ {
			contentArr[i] = arr[i].Content
		}
		content := strings.Join(contentArr, ",,")

		first := arr[0].Content
		t := strings.Split(first, "][")
		eg := ""
		if len(t) >= 3 {
			eg = t[2]
		}

		path, err := api.LinkToSMS(content)
		chat := ""
		if err != nil || path == "" {
			chat = fmt.Sprintf("[P%d][%s] %d %s.  e.g. %s. detail in email", arr[0].Priority, arr[0].Status, size, arr[0].Metric, eg)
			log.Error("create short link fail", err)
		} else {
			chat = fmt.Sprintf("[P%d][%s] %d %s e.g. %s %s/portal/links/%s ",
				arr[0].Priority, arr[0].Status, size, arr[0].Metric, eg, g.Config().Api.Dashboard, path)
			log.Debugf("combined im is:%s", chat)
		}

		redi.WriteIM([]string{arr[0].IM}, chat)
	}
}

/*
读取/queue/user/sms队列中的短信内容，聚合后入库，并发送聚合短信
 */
func combineSms() {
	dtos := popAllSmsDto() // 循环读取/queue/user/sms队列中的短信内容，直到空或出错。以[]*SmsDto的形式返回
	count := len(dtos)
	if count == 0 {
		return
	}

	// 短信聚合，（Priority,Status,Phone,Metric）相同的放到一起
	dtoMap := make(map[string][]*SmsDto)
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("%d%s%s%s", dtos[i].Priority, dtos[i].Status, dtos[i].Phone, dtos[i].Metric)
		if _, ok := dtoMap[key]; ok {
			dtoMap[key] = append(dtoMap[key], dtos[i])
		} else {
			dtoMap[key] = []*SmsDto{dtos[i]}
		}
	}

	for _, arr := range dtoMap {
		size := len(arr)
		if size == 1 {
			// 只有一条，直接发送
			redi.WriteSms([]string{arr[0].Phone}, arr[0].Content)
			continue
		}

		// 把多个sms内容聚合后写入数据库，只给用户提供一个链接
		contentArr := make([]string, size)
		for i := 0; i < size; i++ {
			contentArr[i] = arr[i].Content
		}
		content := strings.Join(contentArr, ",,") // 以",,"分割的多个短信内容

		// 提取endpoint
		first := arr[0].Content
		t := strings.Split(first, "][") // 单条短信格式：[P%d][%s][%s][][%s %s %s %s %s%s%s][O%d %s]
		eg := ""
		if len(t) >= 3 {
			eg = t[2]
		}

		path, err := api.LinkToSMS(content) // 调用POST ip:port/portal/links/store，将聚合的短信内容存入数据库，并返回链接地址
		sms := ""
		// 构造聚合短信内容
		if err != nil || path == "" {
			sms = fmt.Sprintf("[P%d][%s] %d %s.  e.g. %s. detail in email", arr[0].Priority, arr[0].Status, size, arr[0].Metric, eg)
			log.Error("get short link fail", err)
		} else {
			sms = fmt.Sprintf("[P%d][%s] %d %s e.g. %s %s/portal/links/%s ",
				arr[0].Priority, arr[0].Status, size, arr[0].Metric, eg, g.Config().Api.Dashboard, path)
			log.Debugf("combined sms is:%s", sms)
		}

		redi.WriteSms([]string{arr[0].Phone}, sms) // 放入redis待发送队列
	}
}

/*
循环读取/queue/user/sms队列中的短信内容，直到空或出错
以[]*SmsDto的形式返回
 */
func popAllSmsDto() []*SmsDto {
	ret := []*SmsDto{}
	queue := g.Config().Redis.UserSmsQueue // "userSmsQueue": "/queue/user/sms"

	rc := g.RedisConnPool.Get()
	defer rc.Close()

	for {
		reply, err := redis.String(rc.Do("RPOP", queue)) // RPOP Removes and returns the last element of the list stored at key.
		if err != nil {
			if err != redis.ErrNil {
				log.Error("get SmsDto fail", err)
			}
			break
		}

		if reply == "" || reply == "nil" {
			continue
		}

		var smsDto SmsDto
		err = json.Unmarshal([]byte(reply), &smsDto)
		if err != nil {
			log.Error("json unmarshal SmsDto: %s fail: %v", reply, err)
			continue
		}

		ret = append(ret, &smsDto)
	}

	return ret
}

func popAllMailDto() []*MailDto {
	ret := []*MailDto{}
	queue := g.Config().Redis.UserMailQueue // "userMailQueue": "/queue/user/mail"

	rc := g.RedisConnPool.Get()
	defer rc.Close()

	for {
		reply, err := redis.String(rc.Do("RPOP", queue))
		if err != nil {
			if err != redis.ErrNil {
				log.Error("get MailDto fail", err)
			}
			break
		}

		if reply == "" || reply == "nil" {
			continue
		}

		var mailDto MailDto
		err = json.Unmarshal([]byte(reply), &mailDto)
		if err != nil {
			log.Errorf("json unmarshal MailDto: %s fail: %v", reply, err)
			continue
		}

		ret = append(ret, &mailDto)
	}

	return ret
}

func popAllImDto() []*ImDto {
	ret := []*ImDto{}
	queue := g.Config().Redis.UserIMQueue // "userIMQueue": "/queue/user/im"

	rc := g.RedisConnPool.Get()
	defer rc.Close()

	for {
		reply, err := redis.String(rc.Do("RPOP", queue))
		if err != nil {
			if err != redis.ErrNil {
				log.Error("get ImDto fail", err)
			}
			break
		}

		if reply == "" || reply == "nil" {
			continue
		}

		var imDto ImDto
		err = json.Unmarshal([]byte(reply), &imDto)
		if err != nil {
			log.Error("json unmarshal imDto: %s fail: %v", reply, err)
			continue
		}

		ret = append(ret, &imDto)
	}

	return ret
}
