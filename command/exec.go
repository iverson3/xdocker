package command

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"github.com/iverson3/xdocker/util"
)

/**
setns的C代码中已经出现了xdocker_pid 和 xdocker_cmd 这两个key
主要是为了控制是否执行c代码里面的setns
 */

const EnvExecPid = "xdocker_pid"
const EnvExecCmd = "xdocker_cmd"

func ExecContainer(container string, containerCmdArr []string) error {
	exists, containerName, err := util.ContainerIsExists(container)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("container not exists: %s", container)
	}

	// 获取容器进程的PID
	pid, err := util.GetContainerPidByName(containerName)
	if err != nil {
		return err
	}

	// 获取到的pid为空则表示容器进程已停止运行
	if pid == "" {
		return fmt.Errorf("container is not running")
	}
	containerCmdStr := strings.Join(containerCmdArr, " ")

	// 再次执行当前程序
	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 设置环境变量 (当前程序再次被执行的时候由于环境变量设置好了，cgo代码会正常被执行，从而实现进入指定容器进程的NameSpace中)
	err = os.Setenv(EnvExecPid, pid)
	if err != nil {
		return err
	}
	err = os.Setenv(EnvExecCmd, containerCmdStr)
	if err != nil {
		return err
	}

	// 将环境变量传入即将运行的子进程中
	envs, err := util.GetEnvsByPid(pid)
	if err != nil {
		return err
	}
	// 将上面新设置的环境变量和容器进程已有的环境变量合到一起
	cmd.Env = append(os.Environ(), envs...)

	if err = cmd.Run(); err != nil {
		return err
	}
	return nil
}