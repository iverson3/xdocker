package main

import (
	"errors"
	"fmt"
	"os"
	"studygolang/docker/xdocker/cgroups/subsystems"
	"studygolang/docker/xdocker/namespace"
	"studygolang/docker/xdocker/network"
	"studygolang/docker/xdocker/util"

	"github.com/urfave/cli"
	"studygolang/docker/xdocker/command"
)

var initCommand = cli.Command{
	Name:                   "init",
	Usage:                  "init a container",
	Action: func(ctx *cli.Context) error {
		//fmt.Println("Start initiating...")

		// 初始化子进程，即容器进程
		err := command.InitChildProcess()
		if err != nil {
			fmt.Println("ERROR: container.InitProcess() failed, error:", err)
		}
		return err
	},
}

var runCommand = cli.Command{
	Name:                   "run",
	Usage:                  "Create a container with namespace and cgroups limit",
	Flags:                  []cli.Flag{
		&cli.BoolFlag{
			Name:        "it",
			Usage:       "open an interactive tty(pseudo terminal)",
			Required:    false,
		},
		&cli.StringFlag{
			Name:        "m",
			Usage:       "limit the memory",
			Required:    false,
		},
		//&cli.StringFlag{
		//	Name:        "cpu",
		//	//Usage:       "limit the cpu amount",
		//	Usage:       "设置进程在指定的CPU上运行（参数为对应CPU的编号）",
		//},
		&cli.IntFlag{
			Name:        "cpuper",
			Usage:       "limit the cpu percentage",
			Required:    false,
		},
		&cli.StringFlag{
			Name:        "cpushare",
			Usage:       "limit the cpu share",
			Required:    false,
		},
		&cli.StringFlag{
			Name:        "v",
			Usage:       "volume",
			Required:    false,
		},
		&cli.BoolFlag{
			Name:        "d",
			Usage:       "detach container",
			Required:    false,
		},
		&cli.StringFlag{
			Name:        "name",
			Usage:       "container name",
			Required:    false,
		},
		&cli.StringSliceFlag{
			Name:        "e",
			Usage:       "set environment",
			Required:    false,
		},
		&cli.StringFlag{
			Name:        "net",
			Usage:       "container network",
			Required:    false,
		},
		&cli.StringSliceFlag{
			Name:        "p",
			Usage:       "port mapping",
			Required:    false,
		},
	},
	Action: func(ctx *cli.Context) error {
		// 期望的命令格式： ./xdocker run [-name/-v/-d/-it/-m/-cpuper] imageName command
		args := ctx.Args()
		if len(args) < 2 {
			return errors.New("missing image name or command")
		}

		// 镜像名
		imageName := args.Get(0)
		// 检查镜像是否存在
		exist, err := util.ImageIsExist(imageName)
		if err != nil {
			return err
		}
		if !exist {
			return errors.New("image not exist")
		}

		// 收集容器命令
		containerCmd := make([]string, len(args) - 1)
		for index, cmd := range args[1:] {
			containerCmd[index] = cmd
		}

		// 检查是否有参数 "-it"
		tty := ctx.Bool("it")
		// 检查是否有参数 "-d"
		detach := ctx.Bool("d")
		// 获取数据卷
		volume := ctx.String("v")
		// 环境变量
		envSlice := ctx.StringSlice("e")
		// 获取容器名
		containerName := ctx.String("name")
		// 容器网络
		network := ctx.String("net")
		// 端口映射
		portMapping := ctx.StringSlice("p")

		resourceConfig := &subsystems.ResourceConfig{
			MemoryLimit: ctx.String("m"),
			CPUPercentage: ctx.Int("cpuper"),
			CPUShare:    ctx.String("cpushare"),
			//CPUAmount:   ctx.String("cpu"),
		}

		command.Run(tty, detach, containerCmd, resourceConfig, volume, imageName, containerName, envSlice, network, portMapping)
		return nil
	},
}

var listRemoteImageCommand = cli.Command{
	Name:                   "list",
	Usage:                  "list images of image-repository",
	Action: func(ctx *cli.Context) error {
		image := ctx.Args().Get(0)
		return command.ListRemoteImage(image)
	},
}

var searchCommand = cli.Command{
	Name:                   "search",
	Usage:                  "search images of image-repository",
	Action: func(ctx *cli.Context) error {
		args := ctx.Args()
		if len(args) == 0 {
			return errors.New("missing keyword")
		}

		keyword := args.Get(0)
		return command.SearchImage(keyword)
	},
}

var pullCommand = cli.Command{
	Name:                   "pull",
	Usage:                  "pull images from image-repository",
	Action: func(ctx *cli.Context) error {
		args := ctx.Args()
		if len(args) == 0 {
			return errors.New("missing image name")
		}

		image := args.Get(0)
		return command.PullImage(image)
	},
}

var pushCommand = cli.Command{
	Name:                   "push",
	Usage:                  "push images to image-repository",
	Action: func(ctx *cli.Context) error {
		args := ctx.Args()
		if len(args) == 0 {
			return errors.New("missing image name")
		}

		image := args.Get(0)
		return command.PushImage(image)
	},
}

var commitCommand = cli.Command{
	Name:                   "commit",
	Usage:                  "commit a container into image",
	Flags: []cli.Flag {
		&cli.StringFlag{
			Name:        "tag",
			Usage:       "set image tag",
			Required:    true,
		},
	},
	Action: func(ctx *cli.Context) error {
		args := ctx.Args()
		if len(args) == 0 {
			return errors.New("missing container name or container id")
		}

		tag := ctx.String("tag")
		if tag == "" {
			return errors.New("required tag")
		}

		containerFlag := args.Get(0)
		err := command.CommitContainer(containerFlag, tag)
		if err != nil {
			fmt.Println("ERROR: container.CommitContainer() failed, error:", err)
		}
		return err
	},
}

var exportCommand = cli.Command{
	Name:                   "export",
	Usage:                  "export a container into image tar",
	Flags: []cli.Flag {
		&cli.StringFlag{
			Name:        "o",
			Usage:       "export path",
			Required:    false,
		},
	},
	Action: func(ctx *cli.Context) error {
		args := ctx.Args()
		if len(args) == 0 {
			return errors.New("missing container name or container id")
		}

		// 镜像压缩包文件导出路径
		exportPath := ctx.String("o")

		containerFlag := args.Get(0)
		err := command.ExportContainer(containerFlag, exportPath)
		if err != nil {
			fmt.Println("ERROR: container.CommitContainer() failed, error:", err)
		}
		return err
	},
}

var psCommand = cli.Command{
	Name:                   "ps",
	Usage:                  "list all containers",
	Action: func(ctx *cli.Context) error {
		return command.ListContainer()
	},
}

var imagesCommand = cli.Command{
	Name:                   "images",
	Usage:                  "list all images local",
	Action: func(ctx *cli.Context) error {
		return command.ListImages()
	},
}

var removeImageCommand = cli.Command{
	Name:                   "rmi",
	Usage:                  "remove a image local",
	Action: func(ctx *cli.Context) error {
		args := ctx.Args()
		if len(args) == 0 {
			return fmt.Errorf("missing image name")
		}

		// 镜像名
		imageName := args.Get(0)
		return command.RemoveImage(imageName)
	},
}

var buildCommand = cli.Command{
	Name:                   "build",
	Usage:                  "build a image with dockerfile",
	Flags: []cli.Flag {
		&cli.StringFlag{
			Name:        "t",
			Usage:       "output image name and tag",
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "f",
			Usage:       "dockerfile path",
			Required:    false,
		},
	},
	Action: func(ctx *cli.Context) error {
		args := ctx.Args()
		if len(args) == 0 {
			return fmt.Errorf("missing context path")
		}

		// 构建生成的镜像的镜像名和tag (格式：镜像名@tag)
		imageName := ctx.String("t")
		if imageName == "" {
			return fmt.Errorf("required imageName")
		}
		// dockerfile文件路径
		dockerFilePath := ctx.String("f")

		// 上下文路径
		contextPath := args.Get(0)
		return command.BuildImageWithDockerFile(contextPath, imageName, dockerFilePath)
	},
}

var inspectCommand = cli.Command{
	Name:                   "inspect",
	Usage:                  "print a container info",
	Action: func(ctx *cli.Context) error {
		args := ctx.Args()
		if len(args) == 0 {
			return fmt.Errorf("missing container name or container id")
		}

		// 容器名或容器ID
		container := args.Get(0)
		return command.InspectContainer(container)
	},
}


var logCommand = cli.Command{
	Name:                   "logs",
	Usage:                  "print logs of a container",
	Action: func(ctx *cli.Context) error {
		args := ctx.Args()
		if len(args) == 0 {
			return fmt.Errorf("missing container name or container id")
		}

		container := args.Get(0)
		return command.LogContainer(container)
	},
}

var execCommand = cli.Command{
	Name:                   "exec",
	Usage:                  "exec a command into running container",
	Action: func(ctx *cli.Context) error {
		// 第一次执行到这儿的时候由于环境变量还没设置，所以下面的分支不会被执行
		if os.Getenv(command.EnvExecPid) != "" {
			// 第二次被调用执行到这儿的时候，由于环境变量已经设置好了，故会进入该分支
			// 调用namespace包自动调用C代码setns进入容器空间
			namespace.EnterNamespace()
			return nil
		}

		// 命令格式: docker exec 容器名/容器ID 命令
		args := ctx.Args()
		if len(args) < 2 {
			return fmt.Errorf("missing container name/id or command")
		}

		container := args.Get(0)
		containerCmd := make([]string, 0)
		for _, cmd := range args[1:] {
			containerCmd = append(containerCmd, cmd)
		}

		return command.ExecContainer(container, containerCmd)
	},
}

// 暂停容器的运行
var pauseCommand = cli.Command{
	Name:                   "pause",
	Usage:                  "pause a container",
	Action: func(ctx *cli.Context) error {
		args := ctx.Args()
		if len(args) == 0 {
			return fmt.Errorf("missing container name or container id")
		}

		container := args.Get(0)
		return command.PauseContainer(container)
	},
}

// 恢复容器的运行
var continueCommand = cli.Command{
	Name:                   "continue",
	Usage:                  "recover a paused container",
	Action: func(ctx *cli.Context) error {
		args := ctx.Args()
		if len(args) == 0 {
			return fmt.Errorf("missing container name or container id")
		}

		container := args.Get(0)
		return command.RecoverContainer(container)
	},
}

var stopCommand = cli.Command{
	Name:                   "stop",
	Usage:                  "stop a container",
	Action: func(ctx *cli.Context) error {
		args := ctx.Args()
		if len(args) == 0 {
			return fmt.Errorf("missing container name or container id")
		}

		container := args.Get(0)
		return command.StopContainer(container)
	},
}

var startCommand = cli.Command{
	Name:                   "start",
	Usage:                  "start a stopped container",
	Action: func(ctx *cli.Context) error {
		args := ctx.Args()
		if len(args) == 0 {
			return fmt.Errorf("missing container name or container id")
		}

		container := args.Get(0)
		return command.StartContainer(container)
	},
}

var restartCommand = cli.Command{
	Name:                   "restart",
	Usage:                  "restart a container",
	Action: func(ctx *cli.Context) error {
		args := ctx.Args()
		if len(args) == 0 {
			return fmt.Errorf("missing container name or container id")
		}

		container := args.Get(0)
		return command.ReStartContainer(container)
	},
}

var removeCommand = cli.Command{
	Name:                   "rm",
	Usage:                  "remove a container",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:        "f",
			Usage:       "force remove",
			Required:    false,
		},
	},
	Action: func(ctx *cli.Context) error {
		args := ctx.Args()
		if len(args) == 0 {
			return fmt.Errorf("missing container name or container id")
		}

		force := ctx.Bool("f")

		container := args.Get(0)
		return command.RemoveContainer(container, force)
	},
}

// 容器网络相关的操作命令
var networkCommand = cli.Command{
	Name:                   "network",
	Usage:                  "container network commands",
	Subcommands: []cli.Command {
		{
			Name: "create",
			Usage: "create a container network",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name: "driver",
					Usage: "network driver",
					Required: true,
				},
				cli.StringFlag{
					Name: "subnet",
					Usage: "subnet cidr",
					Required: true,
				},
			},
			Action: func(ctx *cli.Context) error {
				args := ctx.Args()
				if len(args) < 1 {
					return fmt.Errorf("missing network name")
				}

				err := network.Init()
				if err != nil {
					return err
				}

				driver := ctx.String("driver")
				subnet := ctx.String("subnet")
				name := args.Get(0)
				return network.CreateNetwork(driver, subnet, name)
			},
		},
		{
			Name: "list",
			Usage: "list container network",
			Action: func(ctx *cli.Context) error {
				err := network.Init()
				if err != nil {
					return err
				}

				return network.ListNetwork()
			},
		},
		{
			Name: "remove",
			Usage: "remove container network",
			Action: func(ctx *cli.Context) error {
				args := ctx.Args()
				if len(args) < 1 {
					return fmt.Errorf("missing network name")
				}

				err := network.Init()
				if err != nil {
					return err
				}

				name := args.Get(0)
				return network.RemoveNetwork(name)
			},
		},
	},
}