package api

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/open-falcon/falcon-plus/modules/alarm/g"
	"github.com/open-falcon/falcon-plus/modules/api/app/model/uic"
	"github.com/toolkits/container/set"
	"github.com/toolkits/net/httplib"
	"strings"
	"sync"
	"time"
)

type APIGetTeamOutput struct {
	uic.Team
	Users       []*uic.User `json:"users"`
	TeamCreator string      `json:"creator_name"`
}

type UsersCache struct {
	sync.RWMutex
	M map[string][]*uic.User
}

var Users = &UsersCache{M: make(map[string][]*uic.User)}

func (this *UsersCache) Get(team string) []*uic.User {
	this.RLock()
	defer this.RUnlock()
	val, exists := this.M[team]
	if !exists {
		return nil
	}

	return val
}

func (this *UsersCache) Set(team string, users []*uic.User) {
	this.Lock()
	defer this.Unlock()
	this.M[team] = users
}

func UsersOf(team string) []*uic.User {
	users := CurlUic(team) // 查询team对应的成员列表

	if users != nil {
		Users.Set(team, users) // 保存到缓存
	} else {
		users = Users.Get(team) // 尝试从缓存读
	}

	return users
}

/*
查询team对应的成员列表，以map的形式返回，key：uic.User.Name; value: *uic.User
 */
func GetUsers(teams string) map[string]*uic.User {
	userMap := make(map[string]*uic.User)
	arr := strings.Split(teams, ",")
	for _, team := range arr {
		if team == "" {
			continue
		}

		users := UsersOf(team) // 查询team对应的成员列表
		if users == nil {
			continue
		}

		for _, user := range users {
			userMap[user.Name] = user
		}
	}
	return userMap
}

/*
根据teams查询到成员列表
将所有成员的phone、mail、im分别将入set，并返回phoneSet,mailSet,imSet
 */
// return phones, emails, IM
func ParseTeams(teams string) ([]string, []string, []string) {
	if teams == "" {
		return []string{}, []string{}, []string{}
	}

	userMap := GetUsers(teams) // 查询team对应的成员列表，以map的形式返回
	phoneSet := set.NewStringSet()
	mailSet := set.NewStringSet()
	imSet := set.NewStringSet()
	for _, user := range userMap { // 遍历userMap，构造phoneSet/mailSet/imSet
		if user.Phone != "" {
			phoneSet.Add(user.Phone)
		}
		if user.Email != "" {
			mailSet.Add(user.Email)
		}
		if user.IM != "" {
			imSet.Add(user.IM)
		}
	}
	return phoneSet.ToSlice(), mailSet.ToSlice(), imSet.ToSlice()
}

/*
GET ip:port/api/v1/team/name/<team>
查询team对应的成员列表
 */
func CurlUic(team string) []*uic.User {
	if team == "" {
		return []*uic.User{}
	}

	uri := fmt.Sprintf("%s/api/v1/team/name/%s", g.Config().Api.PlusApi, team)
	req := httplib.Get(uri).SetTimeout(2*time.Second, 10*time.Second)
	token, _ := json.Marshal(map[string]string{
		"name": "falcon-alarm",
		"sig":  g.Config().Api.PlusApiToken,
	})
	req.Header("Apitoken", string(token))

	var team_users APIGetTeamOutput
	err := req.ToJson(&team_users)
	if err != nil {
		log.Errorf("curl %s fail: %v", uri, err)
		return nil
	}

	return team_users.Users
}
