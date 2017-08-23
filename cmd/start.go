package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/open-falcon/falcon-plus/g"
	"github.com/spf13/cobra"
)

var Start = &cobra.Command{
	Use:   "start [Module ...]",
	Short: "Start Open-Falcon modules",
	Long: `
Start the specified Open-Falcon modules and run until a stop command is received.
A module represents a single node in a cluster.
Modules:
	` + "all " + strings.Join(g.AllModulesInOrder, " "),
	RunE:          start,
	SilenceUsage:  true,
	SilenceErrors: true,
}

var PreqOrderFlag bool
var ConsoleOutputFlag bool
/*
添加配置文件参数
 */
func cmdArgs(name string) []string {
	return []string{"-c", g.Cfg(name)}
}
/*
创建日志文件，返回文件句柄
 */
func openLogFile(name string) (*os.File, error) {
	logDir := g.LogDir(name)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	logPath := g.LogPath(name)
	logOutput, err := os.OpenFile(logPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return logOutput, nil
}

func execModule(co bool, name string) error {
	cmd := exec.Command(g.Bin(name), cmdArgs(name)...) // demo: /bin/falcon-agent -c /etc//agent/config/cfg.json

	// 输出日志信息到console
	if co {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// 输出日志信息到文件
	logOutput, err := openLogFile(name)
	if err != nil {
		return err
	}
	defer logOutput.Close()
	cmd.Stdout = logOutput
	cmd.Stderr = logOutput
	return cmd.Start()
}

/*
判断module和配置文件是否存在
 */
func checkStartReq(name string) error {
	if !g.HasModule(name) {  // 判断是否存在该module
		return fmt.Errorf("%s doesn't exist", name)
	}

	if !g.HasCfg(name) { // 判断配置文件是否存在
		r := g.Rel(g.Cfg(name)) // 返回配置文件相对于当前工作目录的相对路径
		return fmt.Errorf("expect config file: %s", r)
	}

	return nil
}

func isStarted(name string) bool {
	ticker := time.NewTicker(time.Millisecond * 100) // 检查周期
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if g.IsRunning(name) { // 判断进程是否存在
				return true
			}
		case <-time.After(time.Second): // 最大等待时长
			return false
		}
	}
}

func start(c *cobra.Command, args []string) error {
	args = g.RmDup(args)  // args去重

	if PreqOrderFlag {
		args = g.PreqOrder(args) // args排序
	}

	// 不指定args，则默认启动全部模块
	if len(args) == 0 {
		args = g.AllModulesInOrder
	}

	for _, moduleName := range args {
		if err := checkStartReq(moduleName); err != nil { // 判断module和配置文件是否存在
			return err
		}

		// Skip starting if the module is already running
		if g.IsRunning(moduleName) { // 判断进程是否存在
			fmt.Print("[", g.ModuleApps[moduleName], "] ", g.Pid(moduleName), "\n")
			continue
		}

		// 执行模块可执行文件
		if err := execModule(ConsoleOutputFlag, moduleName); err != nil {
			return err
		}

		// 判断模块是否启动成功
		if isStarted(moduleName) {
			fmt.Print("[", g.ModuleApps[moduleName], "] ", g.Pid(moduleName), "\n")
			continue
		}

		return fmt.Errorf("[%s] failed to start", g.ModuleApps[moduleName])
	}
	return nil
}
