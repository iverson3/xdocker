package command

import (
	"fmt"
	"os"
	"strings"
	"studygolang/docker/xdocker/images"
	"studygolang/docker/xdocker/model"
	"studygolang/docker/xdocker/util"
)

func PushImage(image string) error {
	imageName := image
	tag := "latest"
	// imagename@1.2.0
	if strings.Contains(image, "@") {
		imageNameArr := strings.Split(image, "@")
		imageName = imageNameArr[0]
		tag = imageNameArr[1]
	}

	imageList, err := images.GetAllImages()
	if err != nil {
		return err
	}

	var targetImage *model.ImageInfo
	for _, item := range imageList {
		// 寻找本地镜像列表中是否有指定的镜像文件
		if item.Name == imageName && item.TAG == tag {
			targetImage = item
			break
		}
	}

	if targetImage == nil {
		return fmt.Errorf("image not exist")
	}

	// 兼容 xxx@latest.tar 和 xxx.tar
	imageTarPath := fmt.Sprintf("%s%s@%s.tar", model.DefaultImagePath, targetImage.Name, targetImage.TAG)
	exist, err := util.PathExist(imageTarPath)
	if err != nil {
		return err
	}
	if !exist {
		imageTarPath = fmt.Sprintf("%s%s.tar", model.DefaultImagePath, targetImage.Name)
		exist, err = util.PathExist(imageTarPath)
		if err != nil {
			return err
		}
		if !exist {
			return fmt.Errorf("image not exist")
		}
	}

	stat, err := os.Stat(imageTarPath)
	if err != nil {
		return err
	}
	// 镜像文件超过50M 不允许push到服务器
	if stat.Size() > 50 * 1024 * 1024 {
		return fmt.Errorf("image too large")
	}

	err = images.UploadImage(imageTarPath, targetImage.Name, tag)
	if err != nil {
		return err
	}

	fmt.Println("push success")
	return nil
}
