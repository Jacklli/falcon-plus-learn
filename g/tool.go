package g

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)
/*
判断日志文件是否存在
 */
func HasLogfile(name string) bool {
	if _, err := os.Stat(LogPath(name)); err != nil {
		return false
	}
	return true
}

/*
按照AllModulesInOrder，对args进行排序
 */
func PreqOrder(moduleArgs []string) []string {
	if len(moduleArgs) == 0 {
		return []string{}
	}

	var modulesInOrder []string

	// get arguments which are found in the order
	for _, nameOrder := range AllModulesInOrder {
		for _, nameArg := range moduleArgs {
			if nameOrder == nameArg {
				modulesInOrder = append(modulesInOrder, nameOrder)
			}
		}
	}
	// get arguments which are not found in the order
	for _, nameArg := range moduleArgs {
		end := 0
		for _, nameOrder := range modulesInOrder {
			if nameOrder == nameArg {
				break
			}
			end++
		}
		if end == len(modulesInOrder) {
			modulesInOrder = append(modulesInOrder, nameArg)
		}
	}
	return modulesInOrder
}

/*
返回配置文件相对于当前工作目录的相对路径
如当前工作目录/tmp，配置文件路径/etc/open-falcon.conf，则返回../etc/open-falcon.conf
 */
func Rel(p string) string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// filepath.Abs() returns an error only when os.Getwd() returns an error;
	abs, _ := filepath.Abs(p)

	r, err := filepath.Rel(wd, abs)
	if err != nil {
		return ""
	}

	return r
}

/*
判断配置文件是否存在
 */
func HasCfg(name string) bool {
	if _, err := os.Stat(Cfg(name)); err != nil {
		return false
	}
	return true
}

/*
判断是否存在该module
 */
func HasModule(name string) bool {
	return Modules[name]
}

/*
调用pgrep获取进程id，并设置PidOf[name] = pidStr
 */
func setPid(name string) {
	output, _ := exec.Command("pgrep", "-f", ModuleApps[name]).Output()
	pidStr := strings.TrimSpace(string(output))
	PidOf[name] = pidStr
}
/*
获取进程id，保存在PidOf[name]
 */
func Pid(name string) string {
	if PidOf[name] == "<NOT SET>" {
		setPid(name)
	}
	return PidOf[name]
}
/*
判断进程是否存在
 */
func IsRunning(name string) bool {
	setPid(name)
	return Pid(name) != ""
}

/*
 string slice去重
 */
func RmDup(args []string) []string {
	if len(args) == 0 {
		return []string{}
	}
	if len(args) == 1 {
		return args
	}

	ret := []string{}
	isDup := make(map[string]bool)
	for _, arg := range args {
		if isDup[arg] == true {
			continue
		}
		ret = append(ret, arg)
		isDup[arg] = true
	}
	return ret
}
