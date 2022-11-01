package command

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"studygolang/docker/xdocker/model"
	"studygolang/docker/xdocker/util"
)

// 构建镜像过程中，中间容器的临时数据卷的挂载点目录
const defaultMountPoint = "/usr/tempmount/"

type BuildContext struct {
	CmdLines [][]string
	ContextDir string
	CurContainerId string
	// map[容器名]容器ID
	// 容器名 是dockerfile中 FROM命令AS后面的名字
	// 容器ID 是dockerfile中 FROM命令创建的容器的容器ID
	ContainerMap map[string]string
	WorkDir string    // 工作目录
	Args map[string]string   // 存放构建过程中需要用到的全局变量
	Envs map[string]map[string]string // 存放各阶段ENV设置的环境变量
	Volume map[string]map[string]string
}

// 支持的命令
var supportCmd = map[string]DockerfileCmd{
	"FROM": &DockerfileFromCmd{},
	"RUN": &DockerfileRunCmd{},
	"COPY": &DockerfileCopyCmd{},
	"WORKDIR": &DockerfileWorkDirCmd{},
	"ENTRYPOINT": &DockerfileEntryPointCmd{},
	"ENV": &DockerfileEnvCmd{},
	"ARG": &DockerfileArgCmd{},
	//"CMD": {},

	//"ADD": {},
	//"EXPOSE": {},
	//"LABEL": {},
	//"VOLUME": {},
	//"USER": {},
}

type DockerfileCmd interface {
	// FormatCheck 命令的格式检查
	FormatCheck(*BuildContext, []string) ([]string, bool)
	// Exec 执行命令
	Exec(*BuildContext, []string) error
}

type DockerfileFromCmd struct {
}
func (dc *DockerfileFromCmd) FormatCheck(buildCtx *BuildContext, cmdLine []string) ([]string, bool) {
	cmdLine = ReplaceArg(buildCtx, cmdLine)
	// FROM命令的可能格式：
	// FROM golang@v1.16 as builder
	// FROM mysql@v5.7

	if len(cmdLine) != 1 && len(cmdLine) != 3 {
		return nil, false
	}
	if len(cmdLine) == 3 && cmdLine[1] != "as" && cmdLine[1] != "AS" {
		return nil, false
	}

	// 初始化对应FROM下的环境变量map (key是FROM中的镜像名)
	buildCtx.Envs[cmdLine[0]] = make(map[string]string)
	return cmdLine, true
}
func (dc *DockerfileFromCmd) Exec(buildCtx *BuildContext, cmdLine []string) (err error) {
	// FROM命令的可能格式：
	// FROM golang@v1.16 as builder
	// FROM mysql@v5.7
	var imageName string
	var tag string
	if strings.Contains(cmdLine[0], "@") {
		names := strings.Split(cmdLine[0], "@")
		imageName = names[0]
		tag = names[1]
	} else {
		imageName = cmdLine[0]
		tag = "latest"
	}
	image := fmt.Sprintf("%s@%s", imageName, tag)

	// 如果镜像在本地不存在 则尝试从镜像服务拉取镜像
	// 先检查本地是否存在该镜像
	exist, err := util.ImageIsExist(image)
	if err != nil {
		return err
	}
	if !exist {
		// 本地不存在，则尝试从镜像服务拉取
		cmd := exec.Command("xdocker", "pull", image)
		fmt.Println("prepare to pull image from imageHub")
		err = cmd.Run()
		if err != nil {
			return err
		}

		exist, _ = util.ImageIsExist(image)
		if !exist {
			fmt.Println("From Cmd check: image not exist")
			return err
		}
	}

	if len(cmdLine) == 3 {
		// FROM golang@v1.16 as builder
		// 判断是否有容器名重名 （dockerfile中不允许出现FROM as后面存在相同的名字）
		if _, ok := buildCtx.ContainerMap[cmdLine[2]]; ok {
			return fmt.Errorf("duplicate container name")
		}
	}

	// 将上下文目录挂载到容器指定目录中，方便之后的命令需要拷贝文件等操作
	volume := fmt.Sprintf("%s:%s", buildCtx.ContextDir, defaultMountPoint)

	// 组装环境变量的命令部分
	var envs string
	for k, v := range buildCtx.Envs[cmdLine[0]] {
		envs = fmt.Sprintf("%s -e %s=%s", envs, k, v)
	}
	// 去除字符串两边的空格 （注意：命令中不能存在多余的空格，否则容器在启动的时候会失败）
	envs = strings.TrimSpace(envs)

	// ./xdocker run -net xdocker0 -p 8989:8000 -v path1:path2 -name c1 -d tcpserver-alpine@3.1.0 gotcpserver
	// 构建启动容器的命令
	var cmd string
	if envs == "" {
		cmd = fmt.Sprintf("xdocker run -v %s -net %s -d %s top", volume, model.DefaultNetworkName, image)
	} else {
		cmd = fmt.Sprintf("xdocker run -v %s -net %s %s -d %s top", volume, model.DefaultNetworkName, envs, image)
	}
	// 因为上面启动容器采用的是 -d 后台模式，所以cmd.Run()返回了也不能表示容器进程已经完全运行起来了
	cmdResult, err := util.RunCommand(cmd)
	if err != nil {
		return err
	}

	// 检查进程是否完全运行起来
	cmd1 := exec.Command("ps", "-ef")
	var outBuf1 bytes.Buffer
	cmd1.Stdout = &outBuf1
	err = cmd1.Run()
	if err != nil {
		return err
	}

	info, err := util.GetContainerInfoByName(cmdResult[:10])
	if err != nil {
		return err
	}

	cmd2 := exec.Command("grep", info.Pid)
	// 将上个命令执行的标准输出结果作为当前命令的标准输入
	cmd2.Stdin = &outBuf1
	var outBuf2 bytes.Buffer
	cmd2.Stdout = &outBuf2

	err = cmd2.Run()
	if err != nil {
		return err
	}

	// 10位的容器ID+换行符
	if len(cmdResult) != 11 {
		fmt.Printf("cmd run result: %s", cmdResult)
		return fmt.Errorf("container run failed, the run result is not containerId")
	}
	// 取前十位容器ID
	containerId := cmdResult[:10]

	if len(cmdLine) == 3 {
		// 有as 则将as后面的字符串作为其容器名
		buildCtx.ContainerMap[cmdLine[2]] = containerId
	} else {
		// 没有as 则将容器ID作为其容器名
		buildCtx.ContainerMap[containerId] = containerId
	}

	// 更新当前的容器ID
	buildCtx.CurContainerId = containerId

	// 设置容器的挂载信息
	m := make(map[string]string)
	m[buildCtx.ContextDir] = defaultMountPoint
	buildCtx.Volume[containerId] = m

	return nil
}

type DockerfileWorkDirCmd struct {
}
func (dc *DockerfileWorkDirCmd) FormatCheck(buildCtx *BuildContext, cmdLine []string) ([]string, bool) {
	cmdLine = ReplaceArg(buildCtx, cmdLine)
	// workdir一次只能指定一个工作目录
	if len(cmdLine) != 1 {
		return nil, false
	}
	return cmdLine, true
}
func (dc *DockerfileWorkDirCmd) Exec(buildCtx *BuildContext, cmdLine []string) (err error) {
	workDir := cmdLine[0]
	// 在容器中创建该工作目录，不存在才会创建，存在则忽略
	// mkdir -p dirname    -p 创建多级目录并自动忽略已存在的目录
	cmd := fmt.Sprintf("xdocker exec %s mkdir -p %s", buildCtx.CurContainerId, workDir)
	_, err = util.RunCommand(cmd)
	if err != nil {
		return err
	}

	// 将工作目录设置到上下文中，方便后续命令使用
	buildCtx.WorkDir = workDir
	return nil
}

type DockerfileCopyCmd struct {
}
func (dc *DockerfileCopyCmd) FormatCheck(buildCtx *BuildContext, cmdLine []string) ([]string, bool) {
	cmdLine = ReplaceArg(buildCtx, cmdLine)
	// COPY main.go main.go
	// COPY . /go/src/project/
	// COPY . .
	// COPY Gopkg.lock Gopkg.toml /go/src/project/
	// COPY --from=builder /bin/project /bin/project
	if len(cmdLine) < 2 {
		return nil, false
	}

	// todo: 检查--from=builder格式的正确性
	if len(cmdLine) == 3 && strings.Contains(cmdLine[0], "--from") {
		if !strings.Contains(cmdLine[0], "=") || len(cmdLine[0]) <= 7 {
			return nil, false
		}
	}
	return cmdLine, true
}
func (dc *DockerfileCopyCmd) Exec(buildCtx *BuildContext, cmdLine []string) (err error) {
	// 支持复制单个文件或整个目录
	var cmd string
	// 判断是否需要从前面的某个容器往当前容器copy文件
	if strings.Contains(cmdLine[0], "--from=") {
		// 跨容器拷贝文件
		src := cmdLine[1]
		dst := cmdLine[2]

		fromArgs := strings.Split(cmdLine[0], "=")
		containerName := fromArgs[1]
		srcContainerId := buildCtx.ContainerMap[containerName]

		srcContainerRoot := fmt.Sprintf(model.DefaultContainerRoot, srcContainerId)
		srcFullPath := filepath.Join(srcContainerRoot, "mnt", src)

		dstContainerRoot := fmt.Sprintf(model.DefaultContainerRoot, buildCtx.CurContainerId)
		dstFullPath := filepath.Join(dstContainerRoot, "mnt", dst)

		// 判断需要拷贝的是单个文件还是一个目录
		info, err := os.Stat(srcFullPath)
		if err != nil {
			return err
		}
		if info.IsDir() {
			// 如果是要拷贝目录，则dst也必须是一个目录 或者 dst目录不存在则先创建该目录
			exist, err := util.PathExist(dstFullPath)
			if err != nil {
				return err
			}
			if !exist {
				err = os.MkdirAll(dstFullPath, os.ModePerm)
				if err != nil {
					return err
				}
			}

			// 调整源目录路径，使其拷贝源目录下的所有文件 而不是该目录
			if srcFullPath[len(srcFullPath) -1] != '/' {
				srcFullPath = srcFullPath + "/"
			}
			// 后面加"*"表示复制目录下所有文件，加"."表示复制所有文件包括隐藏文件
			srcFullPath = srcFullPath + "."

			cmd = fmt.Sprintf("cp -R %s %s", srcFullPath, dstFullPath)
		} else {
			cmd = fmt.Sprintf("cp %s %s", srcFullPath, dstFullPath)
		}

	} else {
		// 从宿主机往容器中拷贝文件
		src := cmdLine[0]
		dst := cmdLine[1]

		// 宿主机上的目录
		srcFullPathOnHost := filepath.Join(buildCtx.ContextDir, src)
		containerRoot := fmt.Sprintf(model.DefaultContainerRoot, buildCtx.CurContainerId)
		dstFullPathOnHost := filepath.Join(containerRoot, "mnt/", buildCtx.WorkDir, dst)

		fmt.Println(srcFullPathOnHost)
		fmt.Println(dstFullPathOnHost)

		// 容器中的目录
		srcFullPath := filepath.Join(defaultMountPoint, src)
		dstFullPath := filepath.Join(buildCtx.WorkDir, dst)

		// 判断需要拷贝的是单个文件还是一个目录
		info, err := os.Stat(srcFullPathOnHost)
		if err != nil {
			return err
		}
		if info.IsDir() {
			// 如果是要拷贝目录，则dst也必须是一个目录 或者 目标目录不存在则先创建该目录
			//info, err = os.Stat(dstFullPathOnHost)
			//if err != nil {
			//	return err
			//}
			//if !info.IsDir() {
			//	return fmt.Errorf("dst is not directory: %s", dst)
			//}

			// 调整源目录路径，使其拷贝源目录下的所有文件 而不是该目录
			if srcFullPath[len(srcFullPath) -1] != '/' {
				srcFullPath = srcFullPath + "/"
			}
			srcFullPath = srcFullPath + "."

			cmd = fmt.Sprintf("xdocker exec %s cp -R %s %s", buildCtx.CurContainerId, srcFullPath, dstFullPath)
		} else {
			cmd = fmt.Sprintf("xdocker exec %s cp %s %s", buildCtx.CurContainerId, srcFullPath, dstFullPath)
		}
	}

	_, err = util.RunCommand(cmd)
	if err != nil {
		return err
	}
	return nil
}

type DockerfileRunCmd struct {
}
func (dc *DockerfileRunCmd) FormatCheck(buildCtx *BuildContext, cmdLine []string) ([]string, bool) {
	cmdLine = ReplaceArg(buildCtx, cmdLine)
	// 理论上任何在linux下能执行的命令都可以作为Run的参数
	if len(cmdLine) == 0 {
		return nil, false
	}
	return cmdLine, true
}
func (dc *DockerfileRunCmd) Exec(buildCtx *BuildContext, cmdLine []string) (err error) {
	// 对于设置了WORKDIR后执行的命令，都需要使用 && 拼接上 cd命令 切换到对应的工作目录，然后再执行相应的命令
	// cd /usr/go/src/test/ && go build -o /bin/test
	var todoCmd string
	// 判断是否需要切换到对应的工作目录去执行命令
	if buildCtx.WorkDir == "" {
		todoCmd = fmt.Sprintf("%s", strings.Join(cmdLine, " "))
	} else {
		todoCmd = fmt.Sprintf("cd %s && %s", buildCtx.WorkDir, strings.Join(cmdLine, " "))
	}

	// xdocker exec 8995034752 cd /usr/local/ && cat xxx
	cmd := fmt.Sprintf("xdocker exec %s %s", buildCtx.CurContainerId, todoCmd)
	_, err = util.RunCommand(cmd)
	if err != nil {
		return err
	}
	return nil
}

type DockerfileEntryPointCmd struct {
}
func (d DockerfileEntryPointCmd) FormatCheck(buildCtx *BuildContext, cmdLine []string) ([]string, bool) {
	cmdLine = ReplaceArg(buildCtx, cmdLine)
	return cmdLine, true
}
func (d DockerfileEntryPointCmd) Exec(buildCtx *BuildContext, cmdLine []string) error {
	return nil
}

type DockerfileEnvCmd struct {
}
func (d DockerfileEnvCmd) FormatCheck(buildCtx *BuildContext, cmdLine []string) ([]string, bool) {
	cmdLine = ReplaceArg(buildCtx, cmdLine)
	if len(cmdLine) == 0 || len(cmdLine) > 2 {
		return nil, false
	}
	if len(cmdLine) == 1 {
		if !strings.Contains(cmdLine[0], "=") {
			return nil, false
		}
	}

	var imageName string
	// 找到当前ENV处于哪个FROM下
	for _, line := range buildCtx.CmdLines {
		if line[0] == "FROM" {
			imageName = line[1]
		}
		if line[0] == "ENV" && line[1] == cmdLine[0] {
			break
		}
	}

	if imageName == "" {
		return nil, false
	}
	_imageName := imageName

	var tag string
	if strings.Contains(imageName, "@") {
		names := strings.Split(imageName, "@")
		imageName = names[0]
		tag = names[1]
	} else {
		tag = "latest"
	}
	image := fmt.Sprintf("%s@%s", imageName, tag)

	// 如果镜像在本地不存在 则尝试从镜像服务拉取镜像
	// 先检查本地是否存在该镜像
	exist, err := util.ImageIsExist(image)
	if err != nil {
		fmt.Println("Env Cmd check: util.ImageIsExist(image) failed: ", err)
		return nil, false
	}
	if !exist {
		// 本地不存在，则尝试从镜像服务拉取
		cmd := exec.Command("xdocker", "pull", image)
		fmt.Println("prepare to pull image from imageHub")
		err = cmd.Run()
		if err != nil {
			fmt.Println("Env Cmd check: PullImage failed: ", err)
			return nil, false
		}

		exist, _ = util.ImageIsExist(image)
		if !exist {
			fmt.Println("Env Cmd check: image not exist")
			return nil, false
		}
	}

	var envKey string
	var envVal string
	if len(cmdLine) == 1 {
		envArr := strings.Split(cmdLine[0], "=")
		envKey = envArr[0]
		envVal = envArr[1]
	} else {
		envKey = cmdLine[0]
		envVal = cmdLine[1]
	}

	if strings.Contains(envVal, "$PATH") {
		// 获取已有的PATH值 (echo $PATH 获取不到)
		cmd := fmt.Sprintf(`xdocker run -it %s env`, image)
		out, err := util.RunCommand(cmd)
		if err != nil {
			return nil, false
		}

		// 从env的输出中匹配出PATH变量的值
		re := regexp.MustCompile(`PATH=(.*?)\n`)
		findArr := re.FindAllStringSubmatch(out, -1)
		if len(findArr) == 1 {
			path := findArr[0][1]
			envVal = strings.ReplaceAll(envVal, "$PATH", path)
		}
	}

	buildCtx.Envs[_imageName][envKey] = envVal
	return cmdLine, true
}
func (d DockerfileEnvCmd) Exec(buildCtx *BuildContext, cmdLine []string) error {
	// 判断 /etc/bashrc文件是否存在
	//lsCmd := exec.Command("xdocker", "exec", buildCtx.CurContainerId, "ls", "/etc/")
	//var lsOutBuf bytes.Buffer
	//var lsErrBuf bytes.Buffer
	//lsCmd.Stdout = &lsOutBuf
	//lsCmd.Stderr = &lsErrBuf
	//err := lsCmd.Run()
	//fmt.Printf("out: %s\n", lsOutBuf.String())
	//fmt.Printf("err: %s\n", lsErrBuf.String())
	//if err != nil {
	//	return err
	//}
	//grepCmd := exec.Command("grep", "bashrc")
	//var grepOutBuf bytes.Buffer
	//var grepErrBuf bytes.Buffer
	//grepCmd.Stdout = &grepOutBuf
	//grepCmd.Stderr = &grepErrBuf
	//grepCmd.Stdin = &lsOutBuf    // 将上面ls命令的输出作为grep命令的输入
	//err = grepCmd.Run()
	//fmt.Printf("grep out: %s\n", grepOutBuf.String())
	//fmt.Printf("grep err: %s\n", grepErrBuf.String())
	//if err != nil {
	//	fmt.Println("grep: ", err)
	//	return err
	//}

	// 判断/etc/bashrc 文件是否存在
	lsCmd := fmt.Sprintf(`xdocker exec %s sh -c "ls /etc/ | grep bashrc"`, buildCtx.CurContainerId)
	grepRes, err := util.RunCommand(lsCmd)
	if err != nil {
		return err
	}
	if grepRes == "" || grepRes == "\n" || grepRes[:len(grepRes) - 1] != "bashrc" {
		// 不存在/etc/bashrc文件则创建
		touchCmd := fmt.Sprintf("xdocker exec %s touch /etc/bashrc", buildCtx.CurContainerId)
		_, err = util.RunCommand(touchCmd)
		if err != nil {
			return err
		}
	}

	//export PATH=$PATH:/xxx/abc
	//export stefan=12138
	var envKey, envVal string
	if len(cmdLine) == 1 {
		// 其他的环境变量处理
		envArr := strings.Split(cmdLine[0], "=")
		envKey = envArr[0]
		envVal = envArr[1]
	} else {
		if cmdLine[0] == "PATH" || cmdLine[0] == "path" {
			envKey = "PATH"
			envVal = cmdLine[1]
		} else {
			envKey = cmdLine[0]
			envVal = cmdLine[1]
		}
	}

	// 修改/etc/bashrc，在文件末尾追加内容；并通过source命令使其生效
	cmd := fmt.Sprintf(`xdocker exec %s echo "export %s=%s" >> /etc/bashrc && . /etc/bashrc`, buildCtx.CurContainerId, envKey, envVal)
	//cmd := fmt.Sprintf(`xdocker exec %s echo "export %s=%s" >> /etc/profile && source /etc/profile`, buildCtx.CurContainerId, envKey, envVal)
	//cmd := fmt.Sprintf(`xdocker exec %s echo "export %s=%s" >> /etc/profile`, buildCtx.CurContainerId, envKey, envVal)
	_, err = util.RunCommand(cmd)
	if err != nil {
		return err
	}

	sourceCmd := fmt.Sprintf(`xdocker exec %s sh -c "source /etc/bashrc"`, buildCtx.CurContainerId)
	_, err = util.RunCommand(sourceCmd)
	if err != nil {
		return err
	}
	return nil
}

type DockerfileArgCmd struct {
}
func (d DockerfileArgCmd) FormatCheck(buildCtx *BuildContext, cmdLine []string) ([]string, bool) {
	cmdLine = ReplaceArg(buildCtx, cmdLine)
	if len(cmdLine) != 1 {
		return nil, false
	}
	if !strings.Contains(cmdLine[0], "=") {
		return nil, false
	}
	args := strings.Split(cmdLine[0], "=")
	if args[0] == "" || args[1] == "" {
		return nil, false
	}

	buildCtx.Args[args[0]] = strings.Trim(args[1], "\"")
	return cmdLine, true
}
func (d DockerfileArgCmd) Exec(buildCtx *BuildContext, cmdLine []string) error {
	return nil
}

func ReplaceArg(buildCtx *BuildContext, cmdLine []string) []string {
	cmdStr := strings.Join(cmdLine, " ")

	re := regexp.MustCompile(`\$\{(.*?)\}`)
	findArr := re.FindAllStringSubmatch(cmdStr, -1)

	for _, arr := range findArr {
		keyword := arr[1]
		// 从arg中寻找对应的变量
		val, ok := buildCtx.Args[keyword]
		if ok {
			// 将 ${keyword} 整体替换为对应的变量值
			cmdStr = strings.Replace(cmdStr, arr[0], val, 1)
		}
	}

	cmdLine = strings.Split(cmdStr, " ")
	return cmdLine
}

func BuildImageWithDockerFile(contextPath, imageNameTag, dockerFilePath string) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	if dockerFilePath == "" {
		// 如果没有传Dockerfile文件路径，则默认使用当前目录下的Dockerfile文件
		dockerFilePath = filepath.Join(pwd, "Dockerfile")
	}
	exist, err := util.PathExist(dockerFilePath)
	if err != nil {
		return err
	}
	if !exist {
		return fmt.Errorf("dockerfile not exist")
	}

	var imageName, tag string
	if strings.Contains(imageNameTag, "@") {
		imageArr := strings.Split(imageNameTag, "@")
		if len(imageArr) != 2 {
			return fmt.Errorf("imageName format wrong")
		}
		imageName = imageArr[0]
		tag = imageArr[1]
	} else {
		imageName = imageNameTag
		tag = "latest"
	}

	// 判断该镜像是否已经存在
	imageFullPath := fmt.Sprintf("%s%s@%s.tar", model.DefaultImagePath, imageName, tag)
	pathExist, err := util.PathExist(imageFullPath)
	if err != nil {
		return err
	}
	if pathExist {
		return fmt.Errorf("image exist")
	}

	// "."表示当前路径，肯定是存在的；其他路径则需要检查是否存在
	if contextPath != "." {
		exist, err = util.PathExist(contextPath)
		if err != nil {
			return err
		}
		if !exist {
			return fmt.Errorf("context path not exist")
		}
	} else {
		contextPath = pwd
	}

	// 提取dockerfile文件中所有的命令行
	cmdLines, err := scanDockerfile(dockerFilePath)
	if err != nil {
		return err
	}

	// 解析并执行dockerfile中所有的命令
	buildCtx, err := parseDockerfileCommand(contextPath, cmdLines)
	if err != nil {
		fmt.Printf("parseDockerfileCommand failed, error: %v\n", err)
		return err
	}

	// 移除构建过程中的临时挂载点
	err = RemoveMountPoint(buildCtx.Volume)
	if err != nil {
		fmt.Printf("remove tmpMountPoint failed, error: %v\n", err)
		return err
	}

	// 将最终生成的容器打包为镜像文件
	err = ExportContainerToImageTar(buildCtx.CurContainerId, model.DefaultImagePath)
	if err != nil {
		fmt.Printf("export container to image failed, error: %v\n", err)
		return err
	}

	// 将镜像改名为用户设置的镜像名和tag
	err = RenameImage(buildCtx.CurContainerId, imageName, tag)
	if err != nil {
		fmt.Printf("export container to image failed, error: %v\n", err)
	}

	// 清理工作： 移除构建过程中创建的临时容器
	for _, containerId := range buildCtx.ContainerMap {
		err = CleanContainer(containerId)
		if err != nil {
			fmt.Printf("clean container failed, containerId: %s, error: %v\n", containerId, err)
		}
	}

	return nil
}

func parseDockerfileCommand(contextDir string, cmdLines [][]string) (*BuildContext, error) {
	// 构建过程中的上下文，用来保存一些设置的信息
	buildCtx := new(BuildContext)
	buildCtx.ContainerMap = make(map[string]string)
	buildCtx.Volume = make(map[string]map[string]string)
	buildCtx.Args = make(map[string]string)
	buildCtx.Envs = make(map[string]map[string]string)
	buildCtx.ContextDir = contextDir
	buildCtx.CmdLines = cmdLines

	// 依次检查命令格式的正确性
	for i, cmdLine := range cmdLines {
		cmd := supportCmd[cmdLine[0]]
		newCmd, ok := cmd.FormatCheck(buildCtx, cmdLine[1:])
		if !ok {
			return nil, fmt.Errorf("command format check failed, cmd: %v", cmdLine)
		}

		cmdLines[i] = append(cmdLine[:1], newCmd...)
	}

	// 依次执行命令
	for i, cmdLine := range cmdLines {
		fmt.Printf("step %d: %s\n", i+1, strings.Join(cmdLine, " "))
		cmdHead := cmdLine[0]
		cmdBody := cmdLine[1:]

		err := supportCmd[cmdHead].Exec(buildCtx, cmdBody)
		if err != nil {
			// todo: 返回之前要清理容器，将已启动的容器关掉移除等
			// 暂时不处理，方便查看中间结果；可手动remove中间启动的容器
			return nil, err
		}
		fmt.Println("===========================complete=============================")
	}

	// 额外再执行一个命令，创建 /dev/null 文件
	cmd := fmt.Sprintf(`xdocker exec %s touch /dev/null`, buildCtx.CurContainerId)
	_, err := util.RunCommand(cmd)
	if err != nil {
		fmt.Println("Build: touch /dev/null failed: ", err)
	}

	fmt.Println(buildCtx.ContainerMap)

	fmt.Println("build success")
	return buildCtx, nil
}

func RemoveMountPoint(mountPoints map[string]map[string]string) error {
	for containerId, points := range mountPoints {
		for _, point := range points {
			// 拼接出挂载点在宿主机上的目录
			containerRoot := fmt.Sprintf(model.DefaultContainerRoot, containerId)
			mountPointFullPath := filepath.Join(containerRoot, "mnt", point)
			// 取消挂载
			cmd := exec.Command("umount", mountPointFullPath)
			if err := cmd.Run(); err != nil {
				fmt.Println(fmt.Errorf("RemoveMountPoint: umount tmpMountPoint failed, error: %v", err))
				return err
			} else {
				// 删除该临时目录 （删除镜像中的临时挂载目录会导致宿主机当前目录下的文件全部被删掉，原因暂时不明，待完善）
				//if point == defaultMountPoint {
					// 进入容器删除该目录
					//cmd = exec.Command("xdocker", "exec", containerId, "rm", "-rf", point)
					//if err = cmd.Run(); err != nil {
					//	fmt.Println(fmt.Errorf("RemoveMountPoint: rm tmpMountPoint failed, error: %v", err))
					//}
				//}
			}
		}
	}
	return nil
}

// ExportContainerToImageTar 将指定的容器打包为镜像文件并保存到指定的目录下
func ExportContainerToImageTar(containerId string, outPath string) error {
	cmd := exec.Command("xdocker", "export", containerId, "-o", outPath)
	return cmd.Run()
}

// CleanContainer 清理容器
func CleanContainer(containerId string) error {
	// 先停止容器再删除容器
	stopCmd := exec.Command("xdocker", "stop", containerId)
	err := stopCmd.Run()
	if err != nil {
		return err
	}

	rmCmd := exec.Command("xdocker", "rm", containerId)
	err = rmCmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// RenameImage 重命名镜像压缩文件
func RenameImage(containerId, imageName, tag string) error {
	oldPath := fmt.Sprintf("%s%s.tar", model.DefaultImagePath, containerId)
	newPath := fmt.Sprintf("%s%s@%s.tar", model.DefaultImagePath, imageName, tag)
	return os.Rename(oldPath, newPath)
}

// 扫描并获取dockerfile文件中的每一行命令
func scanDockerfile(Dockerfile string) ([][]string, error) {
	f, err := os.Open(Dockerfile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cmdLines := make([][]string, 0)
	reader := bufio.NewReader(f)
	for {
		// 逐行读取文件内容
		line, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}

		// 忽略空行和注释行
		if len(line) != 0 && line[0] != '#' {
			cmdArgs := strings.Split(string(line), " ")
			if len(cmdArgs) == 1 {
				return nil, fmt.Errorf("command format wrong")
			}

			// 检查命令是否支持
			if _, ok := supportCmd[cmdArgs[0]]; !ok {
				return nil, fmt.Errorf("not supported command: %s", cmdArgs[0])
			}

			cmdLines = append(cmdLines, cmdArgs)
		}
	}

	return cmdLines, nil
}