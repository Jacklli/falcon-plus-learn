package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/open-falcon/falcon-plus/g"
	"github.com/spf13/cobra"
)

var Stop = &cobra.Command{
	Use:   "stop [Module ...]",
	Short: "Stop Open-Falcon modules",
	Long: `
Stop the specified Open-Falcon modules.
A module represents a single node in a cluster.
Modules:
  ` + "all " + strings.Join(g.AllModulesInOrder, " "),
	RunE: stop,
}
/*
通过kill -TERM <pid>方式stop
 */
func stop(c *cobra.Command, args []string) error {
	args = g.RmDup(args)  // args去重

	if len(args) == 0 {
		args = g.AllModulesInOrder
	}

	for _, moduleName := range args {
		if !g.HasModule(moduleName) {  // 判断module是否存在
			return fmt.Errorf("%s doesn't exist", moduleName)
		}

		if !g.IsRunning(moduleName) {  // 判断进程是否存在
			fmt.Print("[", g.ModuleApps[moduleName], "] down\n")
			continue
		}

		// 使用kill -TREM <pid>结束进程
		cmd := exec.Command("kill", "-TERM", g.Pid(moduleName))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err == nil {
			fmt.Print("[", g.ModuleApps[moduleName], "] down\n")
			continue
		}
		return err
	}
	return nil
}
