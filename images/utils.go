package images

import (
	"io/ioutil"
	"strconv"
	"strings"
	"github.com/iverson3/xdocker/model"
	"github.com/iverson3/xdocker/util"
)

func GetAllImages() ([]*model.ImageInfo, error) {
	// 遍历 /usr/xdocker/images/ 便可以得到所有的镜像文件
	dirs, err := ioutil.ReadDir(model.DefaultImagePath)
	if err != nil {
		return nil, err
	}

	var images []*model.ImageInfo
	for index, file := range dirs {
		// 去除压缩包文件后缀
		nameArr := strings.Split(file.Name(), ".")
		name := strings.Join(nameArr[:len(nameArr)-1], ".")

		tag := "latest"
		// 从文件名中分离tag
		if strings.Contains(name, "@") {
			nameArr = strings.Split(name, "@")
			tag = nameArr[len(nameArr)-1]
			name = strings.Join(nameArr[:len(nameArr)-1], "@")
		}

		info := &model.ImageInfo{
			ID:         strconv.Itoa(index+1),
			Name:       name,
			Size:       util.FormatFileSize(file.Size()),
			TAG:        tag,
			CreateTime: file.ModTime().Format("2006-01-02 15:04:05"),
		}

		images = append(images, info)
	}
	return images, nil
}

