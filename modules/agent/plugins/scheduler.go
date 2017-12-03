package plugins

import (
	"bytes"
	"encoding/json"
	"github.com/open-falcon/falcon-plus/common/model"
	"github.com/open-falcon/falcon-plus/modules/agent/g"
	"github.com/toolkits/file"
	"github.com/toolkits/sys"
	"log"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

type PluginScheduler struct {
	Ticker *time.Ticker
	Plugin *Plugin
	Quit   chan struct{}
}

func NewPluginScheduler(p *Plugin) *PluginScheduler {
	scheduler := PluginScheduler{Plugin: p}
	scheduler.Ticker = time.NewTicker(time.Duration(p.Cycle) * time.Second)
	scheduler.Quit = make(chan struct{})
	return &scheduler
}
/*
周期性调度执行Plugin，直到channle Quit被关闭
 */
func (this *PluginScheduler) Schedule() {
	go func() {
		for {
			select {
			case <-this.Ticker.C:
				PluginRun(this.Plugin)
			case <-this.Quit:
				this.Ticker.Stop()
				return
			}
		}
	}()
}
/*
通过close(channel)，停止PluginScheduler
 */
func (this *PluginScheduler) Stop() {
	close(this.Quit)
}
/*
运行Plugin（超时处理），发送结果到transfer

type SysProcAttr struct {
        Chroot       string         // Chroot.
        Credential   *Credential    // Credential.
        Ptrace       bool           // Enable tracing.
        Setsid       bool           // Create session.
        Setpgid      bool           // Set process group ID to Pgid, or, if Pgid == 0, to new pid.
        Setctty      bool           // Set controlling terminal to fd Ctty (only meaningful if Setsid is set)
        Noctty       bool           // Detach fd 0 from controlling terminal
        Ctty         int            // Controlling TTY fd
        Foreground   bool           // Place child's process group in foreground. (Implies Setpgid. Uses Ctty as fd of controlling TTY)
        Pgid         int            // Child's process group ID if Setpgid.
        Pdeathsig    Signal         // Signal that the process will get when its parent dies (Linux only)
        Cloneflags   uintptr        // Flags for clone calls (Linux only)
        Unshareflags uintptr        // Flags for unshare calls (Linux only)
        UidMappings  []SysProcIDMap // User ID mappings for user namespaces.
        GidMappings  []SysProcIDMap // Group ID mappings for user namespaces.
        // GidMappingsEnableSetgroups enabling setgroups syscall.
        // If false, then setgroups syscall will be disabled for the child process.
        // This parameter is no-op if GidMappings == nil. Otherwise for unprivileged
        // users this should be set to false for mappings work.
        GidMappingsEnableSetgroups bool
}
cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}的作用：
将子进程的进程组id设置成与进程id一样，方便后面使用kill -<pgid>的方式杀掉整个进程组，如果不设置{Setpgid: true}，则子进程的进程组id继承自父进程，子进程与父进程同属一个进程组

测试demo:
package main

import (
    "fmt"
    "os/exec"
    "syscall"
    "time"
)

func main() {
    cmd := exec.Command("sleep", "5")
    // cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
    start := time.Now()
    time.AfterFunc(3*time.Second, func() {
        pgid, _ := syscall.Getpgid(cmd.Process.Pid)
        fmt.Println(cmd.Process.Pid, pgid)
        cmd.Process.Kill()
    })
    err := cmd.Run()
    fmt.Printf("pid=%d duration=%s err=%s\n", cmd.Process.Pid, time.Since(start), err)
}

输出：
Setpgid: false
[root@bjzw_40_157 tmp]# ./test
22591 22586
pid=22591 duration=3.00078738s err=signal: killed
[root@bjzw_40_157 ~]# ps -elf
0 S root     22586  6999  0  80   0 -   798 wait   09:21 pts/6    00:00:00 ./test
0 S root     22591 22586  0  80   0 - 25227 hrtime 09:21 pts/6    00:00:00 sleep 5

Setpgid: true
[root@bjzw_40_157 tmp]# ./test
25343 25343
pid=25343 duration=3.000684505s err=signal: killed
[root@bjzw_40_157 ~]# ps -elf
0 S root     25338  6999  0  80   0 -   799 wait   09:23 pts/6    00:00:00 ./test
0 S root     25343 25338  0  80   0 - 25227 hrtime 09:23 pts/6    00:00:00 sleep 5

请参考：Go语言中Kill子进程的正确姿势
https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773
http://www.jianshu.com/p/1f3ec2f00b03
 */
func PluginRun(plugin *Plugin) {

	timeout := plugin.Cycle*1000 - 500
	fpath := filepath.Join(g.Config().Plugin.Dir, plugin.FilePath)

	if !file.IsExist(fpath) {
		log.Println("no such plugin:", fpath)
		return
	}

	debug := g.Config().Debug
	if debug {
		log.Println(fpath, "running...")
	}

	cmd := exec.Command(fpath)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Start()
	if debug {
		log.Println("plugin started:", fpath)
	}

	err, isTimeout := sys.CmdRunWithTimeout(cmd, time.Duration(timeout)*time.Millisecond) // 等待cmd执行完成，如果超时，则杀掉cmd所在的进程组

	errStr := stderr.String()
	if errStr != "" {
		logFile := filepath.Join(g.Config().Plugin.LogDir, plugin.FilePath+".stderr.log")
		if _, err = file.WriteString(logFile, errStr); err != nil {
			log.Printf("[ERROR] write log to %s fail, error: %s\n", logFile, err)
		}
	}

	if isTimeout {
		// has be killed
		if err == nil && debug {
			log.Println("[INFO] timeout and kill process", fpath, "successfully")
		}

		if err != nil {
			log.Println("[ERROR] kill process", fpath, "occur error:", err)
		}

		return
	}

	if err != nil {
		log.Println("[ERROR] exec plugin", fpath, "fail. error:", err)
		return
	}

	// exec successfully
	data := stdout.Bytes()
	if len(data) == 0 {
		if debug {
			log.Println("[DEBUG] stdout of", fpath, "is blank")
		}
		return
	}

	var metrics []*model.MetricValue
	err = json.Unmarshal(data, &metrics) // JSON-encoded data -> struct
	if err != nil {
		log.Printf("[ERROR] json.Unmarshal stdout of %s fail. error:%s stdout: \n%s\n", fpath, err, stdout.String())
		return
	}

	g.SendToTransfer(metrics)
}
