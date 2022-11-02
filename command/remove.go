package command

import (
	"fmt"
	"os"
	"github.com/iverson3/xdocker/cgroups"
	"github.com/iverson3/xdocker/container"
	"github.com/iverson3/xdocker/model"
	"github.com/iverson3/xdocker/util"
)

func RemoveContainer(containerFlag string, force bool) error {
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

	if force {
		if info.Status == model.RUNNING {
			// 强制删除但容器处于运行中，则先停止容器
			err = StopContainer(containerName)
			if err != nil {
				fmt.Println(fmt.Errorf("stop container failed, error: %v", err))
				return err
			}
		}
	} else {
		// 非强制删除 只能删除已经停止运行的容器
		if info.Status != model.STOP {
			return fmt.Errorf("don't remove not stopped container")
		}
	}

	// 删除容器时需要处理的事项：
	// 删除容器信息
	// 删除容器id容器名的映射
	// 删除cgroup的相关目录
	// 删除容器工作空间 取消mnt挂载

	// umount对应容器的mnt挂载，删除对应的读写层
	rootUrl, err := util.GetContainerRootPath(info.ID)
	if err != nil {
		return fmt.Errorf("getContainerRootPath failed, containerId: %s, error: %v", info.ID, err)
	}
	mntUrl := rootUrl + "mnt/"
	container.DeleteWorkSpace(rootUrl, mntUrl, info.Volume)

	// 删除对应的cgroup子系统目录
	cGroupPath := fmt.Sprintf("xdocker/%s", info.ID)
	cm := cgroups.NewCgroupManager(cGroupPath)
	err = cm.Destroy()
	if err != nil {
		fmt.Println(fmt.Errorf("remove cgroup directory failed, error: %v", err))
	}

	// 删除储存容器信息的目录
	dirUrl := fmt.Sprintf(model.DefaultInfoLocation, containerName)
	if err = os.RemoveAll(dirUrl); err != nil {
		return fmt.Errorf("remove dir %s failed, error: %v", dirUrl, err)
	}

	// 移除当前容器的容器ID与容器名的映射关系
	err = util.RemoveContainerMapping(info.ID, containerName)
	if err != nil {
		return fmt.Errorf("run: remove containerId - containerName mapping failed, error: %v", err)
	}
	return nil
}
