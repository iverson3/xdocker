package command

import (
	"fmt"
	"os/exec"
	"strings"
	"github.com/iverson3/xdocker/model"
	"github.com/iverson3/xdocker/util"
)

// CommitContainer 将容器打包为镜像，存储到xdocker的镜像目录下
func CommitContainer(container, tag string) error {
	exists, containerName, err := util.ContainerIsExists(container)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("container not exists: %s", container)
	}

	containerInfo, err := util.GetContainerInfoByName(containerName)
	if err != nil {
		return err
	}
	// 获取到的pid为空则表示容器进程已停止运行
	if containerInfo.Pid == "" || containerInfo.Status != model.RUNNING {
		return fmt.Errorf("container is not running")
	}

	rootUrl, err := util.GetContainerRootPath(containerInfo.ID)
	if err != nil {
		return err
	}

	// 镜像名可能包含了tag，需要分离出无tag的镜像名
	imageName := containerInfo.Image
	if strings.Contains(imageName, "@") {
		nameArr := strings.Split(imageName, "@")
		imageName = nameArr[0]
	}

	mntUrl := rootUrl + "mnt/"
	imageTarUrl := fmt.Sprintf("%s%s@%s.tar", model.DefaultImagePath, imageName, tag)
	// 判断该镜像的该tag是否已经存在
	exist, err := util.PathExist(imageTarUrl)
	if err != nil {
		return err
	}
	if exist {
		return fmt.Errorf("duplicate image tag")
	}

	_, err = exec.Command("tar", "-czf", imageTarUrl, "-C", mntUrl, ".").CombinedOutput()
	if err != nil {
		fmt.Println(fmt.Errorf("CommitContainer: tar container failed, error: %v", err))
		return err
	}
	return nil
}