package g

import (
	"bytes"
	"fmt"
	"github.com/toolkits/file"
	"os/exec"
	"strings"
)
/*
调用git rev-parse HEAD查询最新的commitid，作为PluginVersion
 */
func GetCurrPluginVersion() string {
	if !Config().Plugin.Enabled {
		return "plugin not enabled"
	}

	pluginDir := Config().Plugin.Dir
	if !file.IsExist(pluginDir) {
		return "plugin dir not existent"
	}

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = pluginDir // 指定执行命令的工作目录

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return fmt.Sprintf("Error:%s", err.Error())
	}

	return strings.TrimSpace(out.String())
}
