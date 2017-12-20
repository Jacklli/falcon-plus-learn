package api

import (
	"fmt"
	"github.com/open-falcon/falcon-plus/modules/alarm/g"
	"github.com/toolkits/net/httplib"
	"time"
)

/*
调用POST ip:port/portal/links/store
将聚合的短信内容存入数据库，并返回链接
 */
func LinkToSMS(content string) (string, error) {
	uri := fmt.Sprintf("%s/portal/links/store", g.Config().Api.Dashboard)
	req := httplib.Post(uri).SetTimeout(3*time.Second, 10*time.Second)
	req.Body([]byte(content))
	return req.String()
}
