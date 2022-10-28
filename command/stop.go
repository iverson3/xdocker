package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"studygolang/docker/xdocker/model"
	"studygolang/docker/xdocker/network"
	"studygolang/docker/xdocker/util"
	"syscall"
)

// StopContainer 停止一个运行中的容器
func StopContainer(container string) error {
	exists, containerName, err := util.ContainerIsExists(container)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("container not exists")
	}

	info, err := util.GetContainerInfoByName(containerName)
	if err != nil {
		return err
	}
	// stop只能作用于运行中的容器
	if info.Status != model.RUNNING {
		return fmt.Errorf("container not running")
	}

	pid, err := strconv.Atoi(info.Pid)
	if err != nil {
		return err
	}

	// 给容器进程发送kill信号，停止容器进程
	err = syscall.Kill(pid, syscall.SIGTERM)
	if err != nil {
		return err
	}

	// stop  需要释放IP地址
	// pause 不需要释放IP地址
	// 释放当前容器的IP地址占用
	if info.NetworkName != "" && info.IpAddress != "" {
		if err = network.Init(); err != nil {
			fmt.Println(fmt.Errorf("network Init() failed, error: %v", err))
		} else {
			err = network.ReleaseIpAddress(info.NetworkName, info.IpAddress)
			if err != nil {
				fmt.Println(fmt.Errorf("network ReleaseIpAddress failed, error: %v", err))
			}
		}
	}

	// 更新存储的容器信息
	// 更新容器的运行状态，清空容器进程的PID
	info.Status = model.STOP
	info.Pid = ""
	info.IpAddress = ""

	newInfoBytes, err := json.Marshal(info)
	if err != nil {
		return err
	}

	// 将最新的容器信息写入对应的文件中
	dirUrl := fmt.Sprintf(model.DefaultInfoLocation, containerName)
	configPath := dirUrl + model.ConfigName
	err = ioutil.WriteFile(configPath, newInfoBytes, 0622)
	if err != nil {
		return err
	}

	return nil
}
