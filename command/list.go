package command

import (
	"fmt"
	"strings"
	"github.com/iverson3/xdocker/images"
)

func ListRemoteImage(image string) error {
	var imageName = image
	if image != "" && strings.Contains(image, "@") {
		imageNameArr := strings.Split(image, "@")
		imageName = imageNameArr[0]
	}

	list, err := images.FetchImageList(imageName)
	if err != nil {
		return err
	}

	for _, item := range list {
		fmt.Printf("%s    ", item)
	}
	fmt.Println()
	return nil
}