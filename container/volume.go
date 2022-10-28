package container

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"studygolang/docker/xdocker/model"
	"studygolang/docker/xdocker/util"
)

const (
	WriteLayerName = "writeLayer"
)

// NewWorkSpace 创建新的文件工作空间
func NewWorkSpace(rootUrl, imageName, containerName, mntUrl, volume string) error {
	// 创建init只读层
	err := CreateReadOnlyLayer(rootUrl, imageName)
	if err != nil {
		return err
	}
	// 创建读写层
	err = CreateWriteLayer(rootUrl)
	if err != nil {
		return err
	}
	// 创建mnt目录并挂载
	err = CreateMountPoint(rootUrl, imageName, mntUrl, containerName)
	if err != nil {
		return err
	}

	// 检查数据卷
	if volume != "" {
		volumeUrls, err := volumeUrlExtract(volume)
		if err != nil {
			fmt.Println(err)
			return err
		}

		// 挂载volume
		err = MountVolume(mntUrl, volumeUrls)
		if err != nil {
			return fmt.Errorf("NewWorkSpace: mount volume failed, error: %v", err)
		}
	}
	return nil
}

func MountVolume(mntUrl string, volumeUrls []string) error {
	// 创建宿主机文件目录
	parentUrl, containerUrl := volumeUrls[0], filepath.Join(mntUrl, volumeUrls[1])
	exist, err := util.PathExist(parentUrl)
	if err != nil {
		return err
	}
	if !exist {
		// 如果宿主机没有此目录，则创建
		if err = os.MkdirAll(parentUrl, 0777); err != nil {
			return fmt.Errorf("MountVolume: Mkdir parentUrl failed, error: %v", err)
		}
	}
	// 在容器目录中创建挂载点目录
	exist, err = util.PathExist(containerUrl)
	if err != nil {
		return err
	}
	if exist {
		// 如果容器中有该目录，则先删除
		if err = os.RemoveAll(containerUrl); err != nil {
			return fmt.Errorf("MountVolume: remove old containerUrl failed, error: %v", err)
		}
	}
	// 在容器中创建文件夹
	if err = os.MkdirAll(containerUrl, 0777); err != nil {
		return fmt.Errorf("MountVolume: Mkdir containerUrl failed, error: %v", err)
	}
	// 将宿主机的文件目录挂载到容器挂载点
	//dirs := "dirs=" + parentUrl
	// 将一个目录挂载到另一个目录上  mount --bind test1 test2  （如果不加--bind参数 则test1必须是个块设备）
	cmd := exec.Command("mount", "--bind", parentUrl, containerUrl)
	//cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", containerUrl)
	cmd.Stdout = os.Stdout
	// 用缓冲区接收命令执行的错误信息，方便调试定位问题
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	//cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("MountVolume: mount volume failed, error: %v", stderr.String())
	}
	return nil
}

// CreateReadOnlyLayer 通过镜像的压缩包解压并创建镜像文件夹作为只读层
func CreateReadOnlyLayer(rootUrl string, imageName string) error {
	var tag = "latest"
	if strings.Contains(imageName, "@") {
		nameArr := strings.Split(imageName, "@")
		imageName = nameArr[0]
		tag = nameArr[1]
	}
	imageDir := rootUrl + imageName + "/"
	imageTarPath := model.DefaultImagePath + imageName + "@" + tag + ".tar"

	// 判断镜像文件是否存在
	exist, err := util.PathExist(imageTarPath)
	if err != nil {
		return err
	}
	if !exist {
		return fmt.Errorf("CreateReadOnlyLayer: image tar file does not exist: %v", imageTarPath)
	}

	exist, err = util.PathExist(imageDir)
	if err != nil {
		return err
	}
	if !exist {
		// 镜像解压目录不存在则创建
		if err = os.MkdirAll(imageDir, 0777); err != nil {
			return fmt.Errorf("CreateReadOnlyLayer: mkdir imageDir failed, error: %v", err)
		}
	}

	// 将镜像文件解压到对应的目录中作为只读层
	_, err = exec.Command("tar", "-xvf", imageTarPath, "-C", imageDir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("CreateReadOnlyLayer: tar image file failed, error: %v", err)
	}

	return nil
}

// CreateWriteLayer 创建读写层
func CreateWriteLayer(rootUrl string) error {
	writeUrl := rootUrl + WriteLayerName + "/"
	exist, err := util.PathExist(writeUrl)
	if err != nil {
		return err
	}
	if exist {
		// 如果已经存在则需要先删除之前的
		//DeleteWriteLayer(rootUrl)
		return nil
	}

	// 为读写层创建目录
	if err = os.MkdirAll(writeUrl, 0777); err != nil {
		return fmt.Errorf("CreateWriteLayer: create write layer failed, error: %v", err)
	}
	return nil
}

// CreateMountPoint 挂载到容器目录mnt
func CreateMountPoint(rootUrl, imageName, mntUrl, containerName string) error {
	mountPath := mntUrl
	exist, err := util.PathExist(mountPath)
	if err != nil {
		return fmt.Errorf("CreateMountPoint: pathExist(mntUrl) failed, error: %v", err)
	}
	if exist {
		//DeleteMountPoint(mntUrl)
	} else {
		err = os.MkdirAll(mountPath, 0777)
		if err != nil {
			return fmt.Errorf("CreateMountPoint: create mnt directory failed, error: %v", err)
		}
	}

	if strings.Contains(imageName, "@") {
		nameArr := strings.Split(imageName, "@")
		imageName = nameArr[0]
	}
	imageLayerPath := rootUrl + imageName
	containerLayerPath := rootUrl + WriteLayerName

	// 判断当前系统对overlay和aufs的支持情况，根据情况使用对应的联合文件系统进行mount
	support, err := util.IsSupportOverlay()

	if err != nil || !support {
		// overlay不支持则查看aufs是否支持
		support, err = util.IsSupportAufs()
		if err != nil {
			return err
		}
		if !support {
			return fmt.Errorf("not support aufs and overlay")
		}

		// 使用aufs
		// 将读写层目录与镜像只读层目录mount到mnt目录下
		dirs := "dirs=" + containerLayerPath + ":" + imageLayerPath
		cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mountPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("CreateMountPoint: aufs mount mnt failed, error: %v", err)
		}
	} else {
		// 使用overlay
		// 创建overlay的work目录
		workPath := fmt.Sprintf("%swork", rootUrl)
		exist, err = util.PathExist(workPath)
		if err != nil {
			return fmt.Errorf("CreateMountPoint: pathExist(workPath) failed, error: %v", err)
		}
		if !exist {
			err = os.MkdirAll(workPath, 0777)
			if err != nil {
				return fmt.Errorf("CreateMountPoint: create work directory failed, error: %v", err)
			}
		}

		// 将读写层目录与镜像只读层目录mount到mnt目录下
		dirs := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", imageLayerPath, containerLayerPath, workPath)
		cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", dirs, mountPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("CreateMountPoint: overlay mount mnt failed, error: %v", err)
		}
	}

	return nil
}

// DeleteWorkSpace 当容器删除时同时删除工作空间
func DeleteWorkSpace(rootUrl, mntUrl, volume string) {
	// 镜像层的目录不需要删除
	// 在docker中容器被删除后需要删除掉之前创建的只读层与读写层，镜像的目录是不会删除的
	// 在这里镜像的目录就是容器的只读层
	if volume != "" {
		volumeUrls, err := volumeUrlExtract(volume)
		if err != nil {
			fmt.Println(err)
			DeleteMountPoint(mntUrl)
		} else {
			DeleteMountPointWithVolume(mntUrl, volumeUrls)
		}
	} else {
		DeleteMountPoint(mntUrl)
	}

	DeleteWriteLayer(rootUrl)
	// 将整个容器根目录删除 (不止是容器进程运行的roofs目录 即mnt目录)
	DeleteRootPath(rootUrl)
}

func DeleteRootPath(rootUrl string) {
	err := os.RemoveAll(rootUrl)
	if err != nil {
		fmt.Println(fmt.Errorf("DeleteRootPath: remove container rootPath failed, error: %v", err))
	}
}

func DeleteMountPointWithVolume(mntUrl string, volumeUrls []string) {
	// 相比DeleteMountPoint多做了一步：将容器中的volume目录取消挂载
	// 之所以只umount不删除，是因为数据卷是需要持久化保存的，只需要将挂载点卸载即可
	containerUrl := filepath.Join(mntUrl, volumeUrls[1])
	cmd := exec.Command("umount", containerUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println(fmt.Errorf("DeleteMountPointWithVolume: umount containerUrl failed, error: %v", err))
	}
	DeleteMountPoint(mntUrl)
}

// 数据卷的校验函数
func volumeUrlExtract(volume string) ([]string, error) {
	// 数据卷的格式如下：<宿主机目录>:<容器目录>
	volumeAry := strings.Split(volume, ":")
	if len(volumeAry) != 2 || volumeAry[0] == "" || volumeAry[1] == "" {
		return nil, fmt.Errorf("invalid volume: %s", volume)
	}
	return volumeAry, nil
}

// DeleteWriteLayer 删除读写层目录
func DeleteWriteLayer(rootUrl string) {
	writeUrl := rootUrl + WriteLayerName + "/"
	err := os.RemoveAll(writeUrl)
	if err != nil {
		fmt.Println(fmt.Errorf("DeleteWriteLayer: remove writeLayer failed, error: %v", err))
	}
}

// DeleteMountPoint 取消挂载点并删除mnt目录
func DeleteMountPoint(mntUrl string) {
	// 取消mnt目录的挂载
	cmd := exec.Command("umount", mntUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(fmt.Errorf("DeleteMountPoint: umount mnt failed, error: %v", err))
		return
	}

	// 删除mnt目录
	err = os.RemoveAll(mntUrl)
	if err != nil {
		fmt.Println(fmt.Errorf("DeleteMountPoint: remove mnt failed, error: %v", err))
	}
}