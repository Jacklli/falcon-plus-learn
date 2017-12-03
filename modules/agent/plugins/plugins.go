package plugins

type Plugin struct {
	FilePath string
	MTime    int64
	Cycle    int
}

var (
	Plugins              = make(map[string]*Plugin)
	PluginsWithScheduler = make(map[string]*PluginScheduler)
)
/*
更新Plugins，删除不再需要的Plugin
 */
func DelNoUsePlugins(newPlugins map[string]*Plugin) {
	for currKey, currPlugin := range Plugins {
		newPlugin, ok := newPlugins[currKey]
		if !ok || currPlugin.MTime != newPlugin.MTime {
			deletePlugin(currKey)
		}
	}
}
/*
更新Plugins，添加新的Plugin，并进行调度
 */
func AddNewPlugins(newPlugins map[string]*Plugin) {
	for fpath, newPlugin := range newPlugins {
		if _, ok := Plugins[fpath]; ok && newPlugin.MTime == Plugins[fpath].MTime { // 已经存在，且修改时间一致（表示Plugin没有更新）
			continue
		}

		Plugins[fpath] = newPlugin  // 添加新Plugin
		sch := NewPluginScheduler(newPlugin) // 创建PluginScheduler，调度周期由Plugin.Cycle决定
		PluginsWithScheduler[fpath] = sch
		sch.Schedule()  // 周期性调度执行Plugin，直到channle Quit被关闭
	}
}
/*
删除所有Plugin及其PluginScheduler
 */
func ClearAllPlugins() {
	for k := range Plugins {
		deletePlugin(k)
	}
}
/*
停止调度Plugin，删除Plugin及其PluginScheduler
 */
func deletePlugin(key string) {
	v, ok := PluginsWithScheduler[key] // 返回*PluginScheduler
	if ok {
		v.Stop() // 停止PluginScheduler
		delete(PluginsWithScheduler, key) // 清除PluginScheduler
	}
	delete(Plugins, key) // 删除Plugin
}
