package command

import (
	"fmt"
	"os"
	"studygolang/docker/xdocker/model"
	"studygolang/docker/xdocker/util"
)

func RemoveImage(imageName string) error {
	imageFilePath := fmt.Sprintf("%s%s.tar", model.DefaultImagePath, imageName)
	exist, err := util.PathExist(imageFilePath)
	if err != nil {
		return err
	}
	if !exist {
		return fmt.Errorf("image not exist")
	}

	err = os.Remove(imageFilePath)
	if err != nil {
		return err
	}
	return nil
}
