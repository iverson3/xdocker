package command

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"github.com/iverson3/xdocker/model"
	"github.com/iverson3/xdocker/util"
)

// ExportContainer 将容器打包为镜像压缩包文件并导出到指定的目录
func ExportContainer(container, exportPath string) error {
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

	// 如果没有设置导出目录，则导出到当前目录
	if exportPath == "" {
		exportPath, err = os.Getwd()
		if err != nil {
			return err
		}
	} else {
		// 如果不是绝对路径 则获取其绝对路径
		if !filepath.IsAbs(exportPath) {
			exportPath, err = filepath.Abs(exportPath)
			if err != nil {
				return fmt.Errorf("invalid export path, error: %v", err)
			}
		}
	}

	mntUrl := rootUrl + "mnt/"
	imageTarUrl := filepath.Join(exportPath, fmt.Sprintf("%s.tar", containerName))
	_, err = exec.Command("tar", "-czf", imageTarUrl, "-C", mntUrl, ".").CombinedOutput()
	if err != nil {
		fmt.Println(fmt.Errorf("CommitContainer: tar container failed, error: %v", err))
		return err
	}
	return nil
}
