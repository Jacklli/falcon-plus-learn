package cron

import (
	"encoding/json"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
	cmodel "github.com/open-falcon/falcon-plus/common/model"
	"github.com/open-falcon/falcon-plus/modules/alarm/g"
	eventmodel "github.com/open-falcon/falcon-plus/modules/alarm/model/event"
)

/*
处理高优先级报警事件，包括入库、调callback、报警入redis待发送队列
 */
func ReadHighEvent() {
	/*
		    "highQueues": [
	            "event:p0",
	            "event:p1",
	            "event:p2"
	        ],
	*/
	queues := g.Config().Redis.HighQueues
	if len(queues) == 0 {
		return
	}

	for {
		event, err := popEvent(queues) // 从指定队列queues中pop出一个event，存入数据库并返回该event
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		// 调用callback，并发送短信、IM、邮件到redis的待发送队列（高优先级报警不合并）或待合并队列（低优先级报警）
		consume(event, true)
	}
}

/*
处理低优先级报警事件，包括入库、调callback、报警入redis待合并队列
 */
func ReadLowEvent() {
	/*
	    "lowQueues": [
            "event:p3",
            "event:p4",
            "event:p5",
            "event:p6"
        ],
	 */
	queues := g.Config().Redis.LowQueues
	if len(queues) == 0 {
		return
	}

	for {
		event, err := popEvent(queues)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		consume(event, false)
	}
}

/*
从指定队列queues中pop出一个event，存入数据库并返回该event
*/
func popEvent(queues []string) (*cmodel.Event, error) {

	count := len(queues)

	// 构造BRPOP的参数列表
	params := make([]interface{}, count+1)
	for i := 0; i < count; i++ {
		params[i] = queues[i]
	}
	// set timeout 0
	params[count] = 0

	rc := g.RedisConnPool.Get()
	defer rc.Close()

	/*
		BRPOP 是列表的阻塞式(blocking)弹出原语。
		它是 RPOP 命令的阻塞版本，当给定列表内没有任何元素可供弹出的时候，连接将被 BRPOP 命令阻塞，直到等待超时或发现可弹出元素为止。
		当给定多个 key 参数时，按参数 key 的先后顺序依次检查各个列表，弹出第一个非空列表的尾部元素。
		brpop key [key...] timetout

		返回值：
	    假如在指定时间内没有任何元素被弹出，则返回一个 nil 和等待时长。
	    反之，返回一个含有两个元素的列表，第一个元素是被弹出元素所属的 key ，第二个元素是被弹出元素的值。
	*/
	reply, err := redis.Strings(rc.Do("BRPOP", params...))
	if err != nil {
		log.Errorf("get alarm event from redis fail: %v", err)
		return nil, err
	}

	var event cmodel.Event
	err = json.Unmarshal([]byte(reply[1]), &event)
	if err != nil {
		log.Errorf("parse alarm event fail: %v", err)
		return nil, err
	}

	log.Debugf("pop event: %s", event.String())

	//insert event into database
	eventmodel.InsertEvent(&event)
	// events no longer saved in memory

	return &event, nil
}
