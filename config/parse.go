package config

import (
	"bufio"
	"fmt"
	"github.com/iverson3/xdocker/model"
	"github.com/iverson3/xdocker/util"
	"io"
	"os"
	"strings"
)

var (
	// 配置文件路径
	cfgFilePath = "/etc/xdocker.cfg"

	imageHubServerUrlKey = "image_hub_server_host"
	// ImageHubServerUrl xdocker的镜像仓库服务地址
	ImageHubServerUrl = ""

	containerNetworkSubnetKey = "container_network_subnet"
	// ContainerNetworkSubnet 容器的网络子网网段
	ContainerNetworkSubnet = ""
)

// ParseConfig 解析配置文件
func ParseConfig() error {
	defer func() {
		// 使用默认的配置值
		if ImageHubServerUrl == "" {
			ImageHubServerUrl = model.DefaultImageHubServerUrl
		}
		if ContainerNetworkSubnet == "" {
			ContainerNetworkSubnet = model.DefaultNetworkSubnet
		}
	}()

	// 判断配置文件是否存在
	exist, err := util.PathExist(cfgFilePath)
	if err != nil {
		return err
	}

	if exist {
		// 读取配置文件
		cfgFile, err := os.Open(cfgFilePath)
		if err != nil {
			return err
		}

		buf := bufio.NewReader(cfgFile)
		for {
			// 逐行读取文件内容
			line, err := buf.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					// 文件读完了则退出循环
					break
				}
				// 其他错误则直接返回错误
				return err
			}
			line = strings.TrimSpace(line)

			// 检查配置内容格式正确性
			if len(line) < 3 || !strings.Contains(line, "=") {
				return fmt.Errorf("the format of configFile is incorrect")
			}

			lineArr := strings.Split(line, "=")
			if len(lineArr) != 2 {
				return fmt.Errorf("the format of configFile is incorrect")
			}

			key := lineArr[0]
			val := lineArr[1]

			// 去除空格和引号
			key = strings.TrimSpace(key)
			val = strings.TrimSpace(val)
			val = strings.Trim(val, "\"")

			switch key {
			case imageHubServerUrlKey:
				if !strings.Contains(val, "http://") {
					val = fmt.Sprintf("http://%s", val)
				}
				ImageHubServerUrl = val
			case containerNetworkSubnetKey:
				ContainerNetworkSubnet = val
			default:
				// 不支持的配置key
			}
		}
	}
	return nil
}
