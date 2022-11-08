package command

import (
	"fmt"
	"github.com/iverson3/xdocker/model"
	"github.com/iverson3/xdocker/util"
	"os"
	"strings"
)

func RenameImage(oldImageName, newImageName string) error {
	var oldImagePath string
	var newImagePath string
	if strings.Contains(oldImageName, "@") {
		oldImagePath = fmt.Sprintf("%s%s.tar", model.DefaultImagePath, oldImageName)
	} else {
		oldImagePath = fmt.Sprintf("%s%s@latest.tar", model.DefaultImagePath, oldImageName)
	}
	// 判断镜像是否存在
	exist, err := util.PathExist(oldImagePath)
	if err != nil {
		return err
	}
	if !exist {
		return fmt.Errorf("image not exist")
	}

	// 判断新旧镜像名是否一样
	if oldImageName == newImageName {
		return fmt.Errorf("new imageName is the same as old imageName")
	}
	// 根据新镜像名是否包含@ 进行不同的处理
	if strings.Contains(newImageName, "@") {
		imageArr := strings.Split(newImageName, "@")
		// 检查格式是否正确
		if len(imageArr) != 2 || len(imageArr[0]) == 0 || len(imageArr[1]) < 2 || ((imageArr[1])[0] != 'v' && imageArr[1] != "latest") {
			return fmt.Errorf("the format of image name is incorrect")
		}
		newImagePath = fmt.Sprintf("%s%s.tar", model.DefaultImagePath, newImageName)
	} else {
		newImagePath = fmt.Sprintf("%s%s@latest.tar", model.DefaultImagePath, newImageName)
	}

	// 判断新的镜像名是否已存在
	exist, err = util.PathExist(newImagePath)
	if err != nil {
		return err
	}
	if exist {
		return fmt.Errorf("new image name exist")
	}

	return os.Rename(oldImagePath, newImagePath)
}
