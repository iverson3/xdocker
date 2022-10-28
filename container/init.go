package container

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func NewParentProcess(isStart, tty, detach bool, containerId, containerName, imageName, rootUrl, mntUrl, volume string, envSlice []string) (*exec.Cmd, *os.File) {
	// 管道原理和 channel 很像，read 端和 write 端会在另一边没有响应的时候堵塞。
	// 使用 os.Pipe() 获取管道。返回的 readPipe 和 writePipe 都是 *os.File 类型。
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		fmt.Printf("NewParentProcess: new pipe failed, error: %v", err)
		return nil, nil
	}

	// 再次调用自身，第一个命令行参数是 init
	cmd := exec.Command("/proc/self/exe", "init")

	// 命名空间
	// UTS  隔离nodeName和domainName (UTS Namespace)
	// PID  隔离进程 (PID Namespace)
	// IPC  隔离System V IPC和POSIX message queues (IPC Namespace)
	// NET  隔离网络 (Network Namespace)
	// NS   隔离文件系统 (Mount Namespace)
	// USER 隔离用户组ID (User Namespace)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWNET,
		// todo: xxx
		//Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWNET | syscall.CLONE_NEWUSER,
	}

	// 如果设置了交互，就把输出都导入到标准输入输出中 (如果-d后台运行，则输出不能使用标准输出)
	if !detach && tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		// 否则可以将输出写入日志文件中
		logFile, err := CreateLogFile(containerName)
		if err == nil {
			//cmd.Stdin = os.Stdin
			cmd.Stdout = logFile
			cmd.Stderr = logFile
		}
	}

	// 通过环境变量将容器相关信息传递给即将启动的容器进程
	_ = os.Setenv("xdocker_container_id", containerId)
	_ = os.Setenv("xdocker_container_name", containerName)
	_ = os.Setenv("xdocker_volume", volume)

	// 使用 ExtraFile 这个参数将管道(本质也是文件)传给子进程（也就是容器进程）
	// cmd 会带着参数里的文件来创建新的进程
	cmd.ExtraFiles = []*os.File{readPipe}

	// 启动已经被停止的容器是不需要创建工作空间的
	// 重启运行中的容器也是不需要创建工作空间的
	if !isStart {
		// 创建工作空间：包括创建只读层、读写层，联合挂载到mnt目录，进行数据卷的挂载
		err = NewWorkSpace(rootUrl, imageName, containerName, mntUrl, volume)
		if err != nil {
			fmt.Println(fmt.Errorf("NewParentProcess: new workspace failed, error: %v", err))
			return nil, nil
		}
	}

	// 设置进程启动的路径，将mnt目录作为容器的启动目录
	cmd.Dir = mntUrl
	// 环境变量设置，将用户设置的环境变量append进去
	cmd.Env = append(os.Environ(), envSlice...)

	// 把 read 端传给容器进程，然后 write 端保留在父进程中
	return cmd, writePipe
}


