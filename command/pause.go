package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"studygolang/docker/xdocker/model"
	"studygolang/docker/xdocker/util"
	"syscall"
)

// PauseContainer 暂停一个运行中的容器
func PauseContainer(containerFlag string) error {
	exists, containerName, err := util.ContainerIsExists(containerFlag)
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

	// 只能暂停处于运行中的容器
	if info.Status != model.RUNNING {
		return fmt.Errorf("container not running")
	}

	pid, err := strconv.Atoi(info.Pid)
	if err != nil {
		return err
	}

	// 使用"kill -STOP"命令将容器进程暂停
	err = syscall.Kill(pid, syscall.SIGSTOP)
	if err != nil {
		return err
	}

	// 更新容器的状态
	info.Status = model.PAUSED

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
