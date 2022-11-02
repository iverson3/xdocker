package command

import (
	"fmt"
	"github.com/iverson3/xdocker/cgroups"
	"github.com/iverson3/xdocker/cgroups/subsystems"
	"github.com/iverson3/xdocker/container"
	"github.com/iverson3/xdocker/model"
	"github.com/iverson3/xdocker/network"
	"github.com/iverson3/xdocker/util"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

func Run(tty, detach bool, containerCmd []string, res *subsystems.ResourceConfig, volume, imageName, containerName string, envSlice []string, networkName string, portMapping []string) {
	// 是否需要释放资源
	var needRelease = true
	// 生成随机的容器ID
	containerId := util.RandStringBytes(10)
	// 如果没传容器名，则将容器ID作为容器名
	if containerName == "" {
		containerName = containerId
	} else {
		// 检查容器名是否重名
		exists, err := util.ContainerIsExistsByName(containerName)
		if err != nil {
			fmt.Println(fmt.Errorf("unknown error: %v", err))
			return
		}
		if exists {
			fmt.Println("duplicate container name")
			return
		}
	}

	// 不再使用当前路径作为容器运行的根目录，而是使用某个固定的目录+容器ID组成的目录
	rootUrl, err := util.GetContainerRootPath(containerId)
	if err != nil {
		return
	}
	mntUrl := rootUrl + "mnt/"

	// 将新建的只读层和可写层进行隔离
	initProcess, writePipe := container.NewParentProcess(false, tty, detach, containerId, containerName, imageName, rootUrl, mntUrl, volume, envSlice)
	if initProcess == nil || writePipe == nil {
		fmt.Println("new parent process failed")
		// todo: 需要做清理工作，比如删除创建的workspace
		// 但要注意此时workspace可能还没创建 或者 mnt目录还没进行挂载或挂载失败
		// 所以在清理工作之前需要相应的进行判断
		return
	}

	if err := initProcess.Start(); err != nil {
		fmt.Println(fmt.Errorf("ERROR: %v", err))
		// 如果fork进程出现异常，由于mnt已经进行挂载 工作目录已经创建，需要进行清理
		container.DeleteWorkSpace(rootUrl, mntUrl, volume)
		return
	}
	// 自此往后，任何一个步骤出错了，在函数返回之前都要把之前已完成的步骤回滚
	// 回滚处理：释放ip地址 删除容器信息 删除容器id容器名的映射 删除cgroup的相关目录 结束已经运行起来的容器进程 删除容器工作空间 取消mnt挂载
	defer func() {
		if needRelease {
			// 先kill容器进程，再清理容器挂载点和工作空间（镜像层 读写层 mnt）
			// 如果容器不是后台运行，则不需要kill容器进程，因为前台的容器进程退出了代码才会运行到这里，此时容器进程已经退出了，不需要再进行kill
			if detach {
				err = syscall.Kill(initProcess.Process.Pid, syscall.SIGTERM)
				if err != nil {
					fmt.Println(fmt.Errorf("kill container process failed, error: %v", err))
				}
			}

			container.DeleteWorkSpace(rootUrl, mntUrl, volume)
			err = container.RemoveInfoPath(containerName)
			if err != nil {
				fmt.Println(fmt.Errorf("remove container info path failed, error: %v", err))
			}
		}
	}()

	// 将命令参数发送给容器进程
	sendInitCommand(containerCmd, writePipe)

	// 创建资源管理器，进行资源限制的设置
	cGroupPath := fmt.Sprintf(model.DefaultCgroupPath, containerId)
	cm := cgroups.NewCgroupManager(cGroupPath)
	err = cm.Set(res)
	if err != nil {
		fmt.Println(fmt.Errorf("cgroup set resource-limit failed, error: %v", err))
		return
	}
	defer func() {
		if needRelease {
			// 删除限制资源的cgroup子系统相关目录
			err = cm.Destroy()
			if err != nil {
				fmt.Println(fmt.Errorf("remove cgroup directory failed, error: %v", err))
			}
		}
	}()
	err = cm.AddProcess(initProcess.Process.Pid)
	if err != nil {
		fmt.Println(fmt.Errorf("cgroup addProcess failed, error: %v", err))
		return
	}

	// todo: xxx
	//fmt.Println("main process exit")
	//return

	// 容器的网络设置
	var ipAddress string
	if networkName != "" {
		err = network.Init()
		if err != nil {
			fmt.Println(fmt.Errorf("network init failed, error: %v", err))
			return
		} else {
			containerInfo := &model.ContainerInfo{
				Pid:         strconv.Itoa(initProcess.Process.Pid),
				ID:          containerId,
				Name:        containerName,
				PortMapping: portMapping,
			}
			ipAddress, err = network.Connect(networkName, containerInfo)
			if err != nil {
				fmt.Println(fmt.Errorf("network connect failed, network: %s, containerInfo: %v, error: %v", networkName, containerInfo, err))
				return
			}
		}
	}
	if ipAddress != "" {
		defer func() {
			if needRelease {
				// 释放当前容器的IP地址占用
				err = network.ReleaseIpAddress(networkName, ipAddress)
				if err != nil {
					fmt.Println(fmt.Errorf("network.ReleaseIpAddress() failed, ipAddress: %s, error: %v", ipAddress, err))
				}
			}
		}()
	}

	// 记录容器信息
	err = container.RecordContainerInfo(initProcess.Process.Pid, containerCmd, containerId, containerName, imageName, volume, networkName, ipAddress, portMapping)
	if err != nil {
		fmt.Println(fmt.Errorf("run: record container info failed, error: %v", err))
		return
	}
	defer func() {
		if needRelease {
			// 删除容器信息文件
			util.DeleteContainerInfo(containerName)
		}
	}()

	// 记录容器ID与容器名的映射关系
	err = util.AddContainerMapping(containerId, containerName)
	if err != nil {
		fmt.Println(fmt.Errorf("run: add containerId - containerName mapping failed, error: %v", err))
		return
	}
	defer func() {
		if needRelease {
			// 移除当前容器的容器ID与容器名的映射关系
			err = util.RemoveContainerMapping(containerId, containerName)
			if err != nil {
				fmt.Println(fmt.Errorf("run: remove containerId - containerName mapping failed, error: %v", err))
			}
		}
	}()

	//exitCh := make(chan struct{}, 1)
	// 监听退出信号 ctrl+c，但 ctrl+c好像不会执行defer，所以需要将所有相关资源全部释放
	// kill -9 强制杀死进程，信号是直接到达内核，程序是没机会进行资源清理工作的
	//go watchKillSignal(exitCh)

	if !detach {
		// 如果detach为false 则父进程一直等待容器进程的退出
		_ = initProcess.Wait()
		//exitCh <- struct{}{}
		// 非后台容器进程，在容器退出的时候，要删除相关的文件目录  docker是这样做的
		// 而对于后台容器进程，则是在删除容器的时候再删除相关的文件目录
		// 资源释放放在每一步资源设置后的defer中进行
	}

	// detach为true，则父进程直接退出，容器进程成为孤儿进程，让init进程进行接管，由此成为后台进程
	if detach {
		// 容器后台运行则不需要清理资源
		needRelease = false
		fmt.Println(containerId)
	}
	//os.Exit(-1)
	return
}

func watchKillSignal(exitCh chan struct{}) {
	ch := make(chan os.Signal)
	// 监听指定的信号
	signal.Notify(ch, os.Kill, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGABRT)

	// 阻塞直到有信号到来 或者 退出通知的到来
	select {
	case <-exitCh:
		// 停止监听信号
		signal.Stop(ch)
		return
	case <-ch:
		// 停止监听信号
		signal.Stop(ch)
	}
}

func sendInitCommand(containerCmd []string, writePipe *os.File)  {
	cmdString := strings.Join(containerCmd, " ")

	// 向 writePipe 写入参数，这样容器就会获取到参数
	writePipe.WriteString(cmdString)
	// 关闭 pipe，使得 init 进程继续运行
	writePipe.Close()
}

