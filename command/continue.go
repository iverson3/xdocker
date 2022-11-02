package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"github.com/iverson3/xdocker/model"
	"github.com/iverson3/xdocker/util"
	"syscall"
)

// RecoverContainer 恢复一个已暂停的容器，让其继续运行
func RecoverContainer(containerFlag string) error {
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
	if info.Status != model.PAUSED {
		return fmt.Errorf("container not be paused")
	}

	pid, err := strconv.Atoi(info.Pid)
	if err != nil {
		return err
	}

	// 使用"kill -CONT"命令恢复容器进程的运行
	err = syscall.Kill(pid, syscall.SIGCONT)
	if err != nil {
		return err
	}

	// 更新容器的状态
	info.Status = model.RUNNING

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
