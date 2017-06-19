package daemon

import (
	"bytes"
	"fmt"
	"git.oschina.net/kuaishangtong/common/log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	Daemon uint = 1 << iota
	Monitor
)

var (
	once            = new(sync.Once)
	sPid            = strconv.Itoa(os.Getpid())
	sPpid           = strconv.Itoa(os.Getppid())
	env             = os.Environ()
	logPath         = ""
	workerLogPath   = ""
	maxWorkerLogNum = 5
	curWorkerLogNum = 0
)

func SetLogPath(path string) {
	logPath = path
}
func SetWorkerLogPath(path string) {
	workerLogPath = path
}

func Exec(mode uint) {
	once.Do(func() {
		if mode&Daemon == Daemon {
			daemon()
		}
		if mode&Monitor == Monitor {
			monitor()
		}
	})
}

func isDaemoned() bool {
	envPpid := getEnv("__daemon_daemon_ppid__")
	setEnv("__daemon_daemon_ppid__", sPid)
	if sPpid == "1" {
		return true
	}
	if envPpid == sPpid {
		return true
	}
	return false
}

func daemon() {
	// log.Debug("daemon", os.Getpid())
	// time.Sleep(time.Millisecond * 50)
	if isDaemoned() {
		// log.Debug("isDaemoned")
		// time.Sleep(time.Millisecond * 50)
		return
	}
	cmd := getCmd()
	if err := cmd.Start(); err != nil {
		os.Exit(-1)
	}
	os.Exit(0)
}

func isMonitored() bool {
	envPpid := getEnv("__daemon_monitor_ppid__")
	setEnv("__daemon_monitor_ppid__", sPid)
	if envPpid == sPpid {
		return true
	}
	return false
}

func monitor() {
	// log.Debug("monitor", os.Getpid())
	// time.Sleep(time.Millisecond * 50)
	if isMonitored() {
		// log.Debug("isMonitored")
		// time.Sleep(time.Millisecond * 50)
		return
	}
	if logPath == "" {
		logPath = os.Args[0] + ".monitor"
	}
	mlog := log.NewLogger(100000)
	mlog.SetLogFuncCall(true)
	mlog.SetColor(false)
	err := mlog.SetLogFile(logPath, log.LevelInfo, true, 15)
	if err != nil {
		log.Fatal(err)
	}

	var cmd *exec.Cmd
	sigChan := make(chan os.Signal, 4)
	signal.Notify(sigChan)
	stop := false
	go func() {
		for {
			sig := <-sigChan
			if sig == syscall.SIGKILL || sig == syscall.SIGINT || sig == syscall.SIGTERM {
				if cmd != nil && cmd.Process != nil {
					cmd.Process.Signal(sig)
				}
				stop = true
				// os.Exit(0)
			}
		}
	}()
	for {
		if stop {
			os.Exit(0)
		}
		cmd = getCmd()
		if err := cmd.Start(); err != nil {
			os.Exit(-1)
		}
		err := cmd.Wait()
		if err != nil {
			b := cmd.Stderr.(*bytes.Buffer)
			mlog.Warnf("process dead: %s:\n%s", err.Error(), b.String())
			backupWorkerLog()
		} else {
			mlog.Warnf("process dead without error")
		}
		time.Sleep(time.Second)
	}
}

func backupWorkerLog() {
	if workerLogPath != "" {
		prefix := workerLogPath + ".backup."
		num := 0
		fname := ""
		var err error
		for num < maxWorkerLogNum-1 {
			fname = prefix + fmt.Sprintf("%d", num)
			_, err = os.Lstat(fname)
			if err != nil {
				break
			}
			num++
		}
		num = num - 1
		for ; num >= 0; num-- {
			err = os.Rename(prefix+fmt.Sprintf("%d", num), prefix+fmt.Sprintf("%d", num+1))
			if err != nil {
				log.Warnf("Rotate worker log failed: %s", err.Error())
				return
			}
		}

		err = os.Rename(workerLogPath, prefix)
		if err != nil {
			log.Warnf("Backup worker log failed: %s", err.Error())
		}
	}
}

func getCmd() *exec.Cmd {
	cmd := exec.Command(os.Args[0])
	if len(os.Args) > 1 {
		cmd.Args = append(cmd.Args, os.Args[1:]...)
	}
	cmd.Env = env
	b := bytes.NewBuffer(nil)
	cmd.Stderr = b
	return cmd
}

func setEnv(k, v string) {
	k = k + "="
	for i, e := range env {
		if strings.HasPrefix(e, k) {
			env[i] = k + v
			return
		}
	}
	env = append(env, k+v)
}

func getEnv(k string) string {
	k = k + "="
	for _, e := range env {
		if strings.HasPrefix(e, k) {
			return e[len(k):]
		}
	}
	return ""
}

func GetEnv(k string) string {
	return getEnv(k)
}

func SetEnv(k, v string) {
	setEnv(k, v)
}
