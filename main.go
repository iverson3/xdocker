package main

import (
	"fmt"
	"github.com/iverson3/xdocker/model"
	"github.com/iverson3/xdocker/network"
	"github.com/iverson3/xdocker/util"
	"github.com/urfave/cli"
	"os"
	"runtime"
)

func init() {
	err := checkSystemRequire()
	if err != nil {
		panic(err)
	}
	err = initXdockerPath()
	if err != nil {
		panic(err)
	}

	// todo: 检查主机是否能联网
}

func main() {
	app := &cli.App{
		Name: "xDocker",
		Description: "时值 golang 战国年代，冉冉升起的一颗巨星，其名为 XDocker",
		Commands: []cli.Command{
			initCommand,
			runCommand,
			commitCommand,
			listRemoteImageCommand,
			searchCommand,
			pullCommand,
			pushCommand,
			exportCommand,
			imagesCommand,
			removeImageCommand,
			buildCommand,
			psCommand,
			inspectCommand,
			logCommand,
			execCommand,
			pauseCommand,
			continueCommand,
			stopCommand,
			startCommand,
			restartCommand,
			removeCommand,
			networkCommand,
		},
	}

	// 前置处理
	app.Before = func(ctx *cli.Context) error {
		// 判断是否已存在默认的网络，如果不存在则创建该网络 作为容器默认连接的网络
		exists, err := util.CheckNetworkExists(model.DefaultNetworkName)
		if err != nil {
			return err
		}
		if !exists {
			err = network.Init()
			if err != nil {
				return err
			}
			err = network.CreateNetwork(model.DefaultNetworkDriver, model.DefaultNetworkSubnet, model.DefaultNetworkName)
			if err != nil {
				return err
			}
		}

		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}



// 检查对系统的要求是否满足
func checkSystemRequire() error {
	// 判断系统类型 (目前不支持windows)
	if runtime.GOOS == "windows" {
		return fmt.Errorf("not supported on Windows")
	}

	// 判断系统是否支持overlay或aufs
	exist1, err := util.IsSupportOverlay()
	if err != nil {
		return err
	}
	if exist1 {
		return nil
	}
	exist2, err := util.IsSupportAufs()
	if err != nil {
		return err
	}
	if !exist2 {
		return fmt.Errorf("not support aufs and overlay")
	}
	return nil
}

// 初始化xdocker需要用到的各个目录
func initXdockerPath() error {
	// 检查xdocker镜像文件存储目录是否存在，不存在则创建
	exist, err := util.PathExist(model.DefaultImagePath)
	if err != nil {
		return err
	}
	if !exist {
		err = os.MkdirAll(model.DefaultImagePath, 0644)
		if err != nil {
			return err
		}
	}

	// 检查xdocker容器信息存储目录是否存在，不存在则创建
	infoPath := fmt.Sprintf(model.DefaultInfoLocation, "")
	infoPath = infoPath[:len(infoPath)-1]
	exist, err = util.PathExist(infoPath)
	if err != nil {
		return err
	}
	if !exist {
		err = os.MkdirAll(infoPath, 0644)
		if err != nil {
			return err
		}
	}

	// 检查xdocker的cgroup目录是否存在，不存在则创建
	//exist, err = pkg.PathExist(model.DefaultCgroupPath)
	//if err != nil {
	//	return err
	//}
	//if !exist {
	//	err = os.MkdirAll(model.DefaultCgroupPath, 0644)
	//	if err != nil {
	//		return err
	//	}
	//}

	// 检查xdocker的网络信息存储目录是否存在，不存在则创建
	exist, err = util.PathExist(model.DefaultNetworkPath)
	if err != nil {
		return err
	}
	if !exist {
		err = os.MkdirAll(model.DefaultNetworkPath, 0644)
		if err != nil {
			return err
		}
	}

	// 检查xdocker的容器元数据存储目录是否存在，不存在则创建
	exist, err = util.PathExist(model.DefaultMetaDataLocation)
	if err != nil {
		return err
	}
	if !exist {
		err = os.MkdirAll(model.DefaultMetaDataLocation, 0644)
		if err != nil {
			return err
		}
	}

	// 检查xdocker的容器根目录是否存在，不存在则创建
	containerRootPath := fmt.Sprintf(model.DefaultContainerRoot, "")
	containerRootPath = containerRootPath[:len(containerRootPath)-1]
	exist, err = util.PathExist(containerRootPath)
	if err != nil {
		return err
	}
	if !exist {
		err = os.MkdirAll(containerRootPath, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}