package command

import (
	"fmt"
	"strings"
	"github.com/iverson3/xdocker/images"
	"github.com/iverson3/xdocker/util"
)

func PullImage(image string) error {
	// 判断本地是否已存在该镜像
	exist, err := util.ImageIsExist(image)
	if err != nil {
		return err
	}
	if exist {
		return fmt.Errorf("image exist")
	}

	imageName := image
	tag := "latest"
	// imagename@1.2.0
	if strings.Contains(image, "@") {
		imageNameArr := strings.Split(image, "@")
		imageName = imageNameArr[0]
		tag = imageNameArr[1]
	}

	err = images.DownloadImage(imageName, tag)
	if err != nil {
		return err
	}

	fmt.Println("pull success")
	return nil
}
