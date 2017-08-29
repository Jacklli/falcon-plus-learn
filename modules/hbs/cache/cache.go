package cache

import (
	"log"
	"time"
)
/*
周期性加载hostgroup、plugin、host、template、strategy、expression等配置信息到内存

请先阅读以下链接了解以上概念及其关系
https://book.open-falcon.org/zh/usage/getting-started.html
https://book.open-falcon.org/zh/philosophy/tags-and-hostgroup.html
https://github.com/open-falcon-archive/judge#falcon-judge
 */
func Init() {
	log.Println("cache begin")

	log.Println("#1 GroupPlugins...")
	GroupPlugins.Init() // 查询hostgroup id对应的plugin dir，保存到GroupPlugins.M

	log.Println("#2 GroupTemplates...")
	GroupTemplates.Init() // 查询hostgroup id对应的template id，保存到GroupTemplates.M

	log.Println("#3 HostGroupsMap...")
	HostGroupsMap.Init() // 查询hostgroup id对应的host id，保存到HostGroupsMap.M

	log.Println("#4 HostMap...")
	HostMap.Init() // 查询hostsname对应的host id，保存到HostMap.M

	log.Println("#5 TemplateCache...")
	TemplateCache.Init() // 查询所有template信息，保存到TemplateCache.M

	log.Println("#6 Strategies...")
	Strategies.Init(TemplateCache.GetMap()) // 查询所有active（根据run_begin和run_end）的策略信息，保存到SafeStrategies.M

	log.Println("#7 HostTemplateIds...")
	HostTemplateIds.Init() // 查询host id对应的template id，保存到HostTemplateIds.M

	log.Println("#8 ExpressionCache...")
	ExpressionCache.Init() // 查询所有active的Expression，保存到ExpressionCache.L

	log.Println("#9 MonitoredHosts...")
	MonitoredHosts.Init() // 查询所有active的host，保存到MonitoredHosts.M

	log.Println("cache done")

	go LoopInit() // 周期性调用上述Init()

}

func LoopInit() {
	for {
		time.Sleep(time.Minute)
		GroupPlugins.Init()
		GroupTemplates.Init()
		HostGroupsMap.Init()
		HostMap.Init()
		TemplateCache.Init()
		Strategies.Init(TemplateCache.GetMap())
		HostTemplateIds.Init()
		ExpressionCache.Init()
		MonitoredHosts.Init()
	}
}
