package command

import (
	"encoding/json"
	"fmt"
	"github.com/iverson3/xdocker/cgroups"
	"github.com/iverson3/xdocker/container"
	"github.com/iverson3/xdocker/model"
	"github.com/iverson3/xdocker/network"
	"github.com/iverson3/xdocker/util"
	"io/ioutil"
	"strconv"
	"strings"
	"syscall"
)

// StartContainer 启动一个已经被停止的容器
func StartContainer(containerFlag string) error {
	var needRelease = true
	exists, containerName, err := util.ContainerIsExists(containerFlag)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("container not exists: %s", containerFlag)
	}

	info, err := util.GetContainerInfoByName(containerName)
	if err != nil {
		return err
	}

	// 不再使用当前路径作为容器运行的根目录，而是使用某个固定的目录+容器ID组成的目录
	rootUrl, err := util.GetContainerRootPath(info.ID)
	if err != nil {
		return err
	}
	mntUrl := rootUrl + "mnt/"

	// todo: 将envSlice放入容器信息中存储起来
	envSlice := []string{""}

	// 将新建的只读层和可写层进行隔离
	initProcess, writePipe := container.NewParentProcess(true, false, true, info.ID, containerName, info.Image, rootUrl, mntUrl, info.Volume, envSlice)
	if initProcess == nil || writePipe == nil {
		fmt.Println("new parent process failed")
		return fmt.Errorf("new parent process failed")
	}

	if err := initProcess.Start(); err != nil {
		fmt.Println(fmt.Errorf("ERROR: %v", err))
		return err
	}
	defer func() {
		if needRelease {
			// kill容器进程
			err = syscall.Kill(initProcess.Process.Pid, syscall.SIGTERM)
			if err != nil {
				fmt.Println(fmt.Errorf("kill container process failed, error: %v", err))
			}
		}
	}()

	// 将命令参数发送给容器进程
	containerCmd := strings.Split(info.Command, " ")
	sendInitCommand(containerCmd, writePipe)

	// 向对应的资源管理器中加入新起的容器进程Pid
	cGroupPath := fmt.Sprintf(model.DefaultCgroupPath, info.ID)
	cm := cgroups.NewCgroupManager(cGroupPath)
	err = cm.AddProcess(initProcess.Process.Pid)
	if err != nil {
		fmt.Println(fmt.Errorf("cgroup addProcess failed, error: %v", err))
		return err
	}

	// 容器的网络设置
	var ipAddress string
	if info.NetworkName != "" {
		err = network.Init()
		if err != nil {
			fmt.Println(fmt.Errorf("network init failed, error: %v", err))
			return err
		} else {
			containerInfo := &model.ContainerInfo{
				Pid:         strconv.Itoa(initProcess.Process.Pid),
				ID:          info.ID,
				Name:        containerName,
				PortMapping: info.PortMapping,
			}
			ipAddress, err = network.Connect(info.NetworkName, containerInfo)
			if err != nil {
				fmt.Println(fmt.Errorf("network connect failed, network: %s, containerInfo: %v, error: %v", info.NetworkName, containerInfo, err))
				return err
			}
		}
	}
	if ipAddress != "" {
		defer func() {
			if needRelease {
				// 释放为容器分配的IP地址
				err = network.ReleaseIpAddress(info.NetworkName, ipAddress)
				if err != nil {
					fmt.Println(fmt.Errorf("network.ReleaseIpAddress() failed, ipAddress: %s, error: %v", ipAddress, err))
				}
			}
		}()
	}

	// 修改容器信息 - 主要是将新的容器进程Pid和网络IP写入容器信息文件中
	err = updateContainerInfoForStart(info.Name, initProcess.Process.Pid, ipAddress)
	if err != nil {
		fmt.Println(fmt.Errorf("run: record container info failed, error: %v", err))
		return err
	}

	needRelease = false
	return nil
}

func updateContainerInfoForStart(containerName string, pid int, ipAddress string) error {
	info, err := util.GetContainerInfoByName(containerName)
	if err != nil {
		return err
	}

	info.Pid = strconv.Itoa(pid)
	info.IpAddress = ipAddress
	info.Status = model.RUNNING

	infoBytes, err := json.Marshal(info)
	if err != nil {
		return err
	}

	// 将最新的容器信息写入对应的文件中
	dirUrl := fmt.Sprintf(model.DefaultInfoLocation, containerName)
	configPath := dirUrl + model.ConfigName
	err = ioutil.WriteFile(configPath, infoBytes, 0622)
	if err != nil {
		return err
	}

	return nil
}
