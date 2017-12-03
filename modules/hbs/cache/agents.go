package cache

// 每个agent心跳上来的时候立马更新一下数据库是没必要的
// 缓存起来，每隔一个小时写一次DB
// 提供http接口查询机器信息，排查重名机器的时候比较有用

import (
	"github.com/open-falcon/falcon-plus/common/model"
	"github.com/open-falcon/falcon-plus/modules/hbs/db"
	"sync"
	"time"
)

type SafeAgents struct {
	sync.RWMutex
	M map[string]*model.AgentUpdateInfo
}

var Agents = NewSafeAgents()
/*
C不允许这样调用，会报"error: initializer element is not constant"

# cat 1.c
#include<stdio.h>

int foo() {
return 1;
}

int f = foo();

main()
{
printf("Hello World: %d\n", f);
}
# gcc 1.c
1.c:9: error: initializer element is not constant

from https://stackoverflow.com/questions/3025050/error-initializer-element-is-not-constant-when-trying-to-initialize-variable-w
  > In C language objects with static storage duration have to be initialized with constant expressions
  > or with aggregate initializers containing constant expressions.
 */

func NewSafeAgents() *SafeAgents {
	return &SafeAgents{M: make(map[string]*model.AgentUpdateInfo)}
}
/*
更新数据库中的host表和内存中的Agents信息
 */
func (this *SafeAgents) Put(req *model.AgentReportRequest) {
	val := &model.AgentUpdateInfo{
		LastUpdate:    time.Now().Unix(),
		ReportRequest: req,
	}

	if agentInfo, exists := this.Get(req.Hostname); !exists ||
		agentInfo.ReportRequest.AgentVersion != req.AgentVersion ||
		agentInfo.ReportRequest.IP != req.IP ||
		agentInfo.ReportRequest.PluginVersion != req.PluginVersion {

		db.UpdateAgent(val) // insert或者update host表
		this.Lock()
		this.M[req.Hostname] = val // 更新内存中的Agents信息
		this.Unlock()
	}
}
/*
返回hostname对应的host的AgentUpdateInfo
 */
func (this *SafeAgents) Get(hostname string) (*model.AgentUpdateInfo, bool) {
	this.RLock()
	defer this.RUnlock()
	val, exists := this.M[hostname]
	return val, exists
}

func (this *SafeAgents) Delete(hostname string) {
	this.Lock()
	defer this.Unlock()
	delete(this.M, hostname)
}
/*
返回全局变量Agents.M的keys，即host列表
 */
func (this *SafeAgents) Keys() []string {
	this.RLock()
	defer this.RUnlock()
	count := len(this.M)
	keys := make([]string, count)
	i := 0
	for hostname := range this.M {
		keys[i] = hostname
		i++
	}
	return keys
}
/*
每天运行一次，删除内存中超过一天没有心跳的agent
 */
func DeleteStaleAgents() {
	duration := time.Hour * time.Duration(24) // 每天运行一次
	for {
		time.Sleep(duration)
		deleteStaleAgents() // 删除内存中超过一天没有心跳的agent
	}
}

func deleteStaleAgents() {
	// 一天都没有心跳的Agent，从内存中干掉
	before := time.Now().Unix() - 3600*24
	keys := Agents.Keys() // 返回全局变量Agents.M的keys，即host列表
	count := len(keys)
	if count == 0 {
		return
	}

	for i := 0; i < count; i++ {
		curr, _ := Agents.Get(keys[i]) // 返回对应的host的AgentUpdateInfo
		if curr.LastUpdate < before { // 如果超过一天没有更新，则删除该agent
			Agents.Delete(curr.ReportRequest.Hostname)
		}
	}
}
