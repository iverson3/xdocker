package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"github.com/iverson3/xdocker/model"
	"time"
)

// GetContainerRootPath 获取容器的根目录
func GetContainerRootPath(containerId string) (string, error) {
	rootUrl := fmt.Sprintf(model.DefaultContainerRoot, containerId)
	// 检查xdocker容器根目录是否存在，否则创建目录
	exist, err := PathExist(rootUrl)
	if err != nil {
		return "", err
	}
	if !exist {
		// 目录不存在则创建
		if err = os.MkdirAll(rootUrl, 0777); err != nil {
			return "", err
		}
	}

	return rootUrl, nil
}

func GetContainerInfoByName(containerName string) (*model.ContainerInfo, error) {
	dirUrl := fmt.Sprintf(model.DefaultInfoLocation, containerName)
	configPath := dirUrl + model.ConfigName

	// 先判断容器信息存储文件是否存在，如果不存在则说明指定容器不存在
	exist, err := PathExist(configPath)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, fmt.Errorf("container is not exist")
	}

	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	info := new(model.ContainerInfo)
	err = json.Unmarshal(content, info)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func ContainerIsExistsByName(containerName string) (bool, error) {
	// 遍历 /var/run/xdocker 便可以得到所有的容器目录，容器目录名就是容器名
	dirUrl := fmt.Sprintf(model.DefaultInfoLocation, "")
	dirUrl = dirUrl[:len(dirUrl)-1]

	dirs, err := ioutil.ReadDir(dirUrl)
	if err != nil {
		return false, err
	}

	// 遍历所有的容器目录
	for _, dir := range dirs {
		if dir.Name() == containerName {
			return true, nil
		}
	}
	return false, nil
}

// ContainerIsExists 通过容器ID或容器名判断容器是否存在，存在则统一返回容器名
func ContainerIsExists(container string) (bool, string, error) {
	// 该文件中存储了所有容器的对应关系 (容器名->容器ID)
	path1 := fmt.Sprintf("%s%s", model.DefaultMetaDataLocation, "containers.name.map")
	exist, err := PathExist(path1)
	if err != nil {
		return false, "", err
	}
	if !exist {
		return false, "", nil
	}

	bytes, err := ioutil.ReadFile(path1)
	if err != nil {
		return false, "", err
	}
	if len(bytes) == 0 || len(bytes) == 2 {
		return false, "", nil
	}

	// 容器名 -> 容器ID 的映射
	name2idMapping := make(map[string]string)
	err = json.Unmarshal(bytes, &name2idMapping)
	if err != nil {
		return false, "", err
	}

	if _, ok := name2idMapping[container]; ok {
		// 能在 name2idMapping 里找到说明容器存在 并且 container 就是容器名
		return true, container, nil
	}

	// 该文件中存储了所有容器的对应关系 (容器ID->容器名)
	path2 := fmt.Sprintf("%s%s", model.DefaultMetaDataLocation, "containers.id.map")
	exist, err = PathExist(path2)
	if err != nil {
		return false, "", err
	}
	if !exist {
		return false, "", nil
	}

	bytes, err = ioutil.ReadFile(path2)
	if err != nil {
		return false, "", err
	}
	if len(bytes) == 0 || len(bytes) == 2 {
		return false, "", nil
	}

	// 容器ID -> 容器名 的映射
	id2nameMapping := make(map[string]string)
	err = json.Unmarshal(bytes, &id2nameMapping)
	if err != nil {
		return false, "", err
	}

	if val, ok := id2nameMapping[container]; ok {
		// 能在 id2nameMapping 里找到说明容器存在 并且 container 就是容器ID
		return true, val, nil
	} else {
		// 否则不存在该容器ID或容器名
		return false, "", nil
	}
}

func AddContainerMapping(containerId, containerName string) error {
	// 该文件中存储了所有容器的对应关系 (容器名->容器ID)
	path1 := fmt.Sprintf("%s%s", model.DefaultMetaDataLocation, "containers.name.map")
	exist, err := PathExist(path1)
	if err != nil {
		return err
	}

	// 容器名 -> 容器ID 的映射
	name2idMapping := make(map[string]string)
	var file1 *os.File
	if !exist {
		// 判断目录是否存在
		dirExists, err := PathExist(model.DefaultMetaDataLocation)
		if err != nil {
			return err
		}
		if !dirExists {
			// 目录不存在则创建
			err = os.MkdirAll(model.DefaultMetaDataLocation, 0777)
			if err != nil {
				return err
			}
		}

		// 不存在该文件则创建
		file1, err = os.OpenFile(path1, os.O_RDWR | os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer file1.Close()
	} else {
		file1, err = os.OpenFile(path1, os.O_RDWR, 0644)
		if err != nil {
			return err
		}
		defer file1.Close()
		bytes, err := ioutil.ReadAll(file1)
		if err != nil {
			return err
		}
		if len(bytes) == 0 {
			return fmt.Errorf("empty file: %s", path1)
		}

		err = json.Unmarshal(bytes, &name2idMapping)
		if err != nil {
			return err
		}
		if _, ok := name2idMapping[containerName]; ok {
			return fmt.Errorf("duplicate container name")
		}
		// 清空文件内容，并将写入位置移动到文件起始位置
		_ = file1.Truncate(0)
		_, _ = file1.Seek(0, 0)
	}

	name2idMapping[containerName] = containerId

	dataBytes, err := json.Marshal(name2idMapping)
	if err != nil {
		return err
	}

	_, err = file1.Write(dataBytes)
	if err != nil {
		return err
	}


	// 该文件中存储了所有容器的对应关系 (容器ID->容器名)
	path2 := fmt.Sprintf("%s%s", model.DefaultMetaDataLocation, "containers.id.map")
	exist, err = PathExist(path2)
	if err != nil {
		return err
	}

	// 容器ID -> 容器名 的映射
	id2nameMapping := make(map[string]string)
	var file2 *os.File
	if !exist {
		// 不存在该文件则创建
		file2, err = os.OpenFile(path2, os.O_RDWR | os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer file2.Close()
	} else {
		file2, err = os.OpenFile(path2, os.O_RDWR, 0644)
		if err != nil {
			return err
		}
		defer file2.Close()
		bytes, err := ioutil.ReadAll(file2)
		if err != nil {
			return err
		}
		if len(bytes) == 0 {
			return fmt.Errorf("empty file: %s", path2)
		}

		err = json.Unmarshal(bytes, &id2nameMapping)
		if err != nil {
			return err
		}
		if _, ok := id2nameMapping[containerId]; ok {
			return fmt.Errorf("duplicate container name or container id")
		}
		// 清空文件内容，并将写入位置移动到文件起始位置
		_ = file2.Truncate(0)
		_, _ = file2.Seek(0, 0)
	}

	id2nameMapping[containerId] = containerName

	dataBytes, err = json.Marshal(id2nameMapping)
	if err != nil {
		return err
	}

	_, err = file2.Write(dataBytes)
	if err != nil {
		return err
	}

	return nil
}

func RemoveContainerMapping(containerId, containerName string) error {
	// 该文件中存储了所有容器的对应关系 (容器名->容器ID)
	path1 := fmt.Sprintf("%s%s", model.DefaultMetaDataLocation, "containers.name.map")
	exist, err := PathExist(path1)
	if err != nil {
		return err
	}
	if !exist {
		// 不存在该文件则说明有问题
		return fmt.Errorf("file not exist: %s", path1)
	}

	file1, err := os.OpenFile(path1, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file1.Close()
	bytes, err := ioutil.ReadAll(file1)
	if err != nil {
		return err
	}
	if len(bytes) == 0 {
		return fmt.Errorf("empty file: %s", path1)
	}
	if len(bytes) == 2 {
		return fmt.Errorf("no container can be removed")
	}

	// 容器名 -> 容器ID 的映射
	name2idMapping := make(map[string]string)
	err = json.Unmarshal(bytes, &name2idMapping)
	if err != nil {
		return err
	}
	if _, ok := name2idMapping[containerName]; ok {
		// key在map中存在才需要delete 并修改文件内容，否则直接忽略
		delete(name2idMapping, containerName)

		dataBytes, err := json.Marshal(name2idMapping)
		if err != nil {
			return err
		}
		// 写之前 先清空文件内容，并将写入位置移动到文件起始位置
		_ = file1.Truncate(0)
		_, _ = file1.Seek(0, 0)
		_, err = file1.Write(dataBytes)
		if err != nil {
			return err
		}
	}


	// 该文件中存储了所有容器的对应关系 (容器ID->容器名)
	path2 := fmt.Sprintf("%s%s", model.DefaultMetaDataLocation, "containers.id.map")
	exist, err = PathExist(path2)
	if err != nil {
		return err
	}
	if !exist {
		// 不存在该文件则说明有问题
		return fmt.Errorf("file not exist: %s", path2)
	}

	file2, err := os.OpenFile(path2, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file2.Close()
	bytes, err = ioutil.ReadAll(file2)
	if err != nil {
		return err
	}
	if len(bytes) == 0 {
		return fmt.Errorf("empty file: %s", path2)
	}
	if len(bytes) == 2 {
		return fmt.Errorf("no container can be removed")
	}

	// 容器ID -> 容器名 的映射
	id2nameMapping := make(map[string]string)
	err = json.Unmarshal(bytes, &id2nameMapping)
	if err != nil {
		return err
	}
	if _, ok := id2nameMapping[containerId]; ok {
		// key在map中存在才需要delete 并修改文件内容，否则直接忽略
		delete(id2nameMapping, containerId)

		dataBytes, err := json.Marshal(id2nameMapping)
		if err != nil {
			return err
		}
		// 写之前 先清空文件内容，并将写入位置移动到文件起始位置
		_ = file2.Truncate(0)
		_, _ = file2.Seek(0, 0)
		_, err = file2.Write(dataBytes)
		if err != nil {
			return err
		}
	}

	return nil
}

func PathExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func GetContainerPidByName(containerName string) (string, error) {
	containerInfo, err := GetContainerInfoByName(containerName)
	if err != nil {
		return "", err
	}

	return containerInfo.Pid, nil
}

func GetContainerIdByName(containerName string) (string, error) {
	containerInfo, err := GetContainerInfoByName(containerName)
	if err != nil {
		return "", err
	}

	return containerInfo.ID, nil
}

func GetEnvsByPid(pid string) ([]string, error) {
	// 读取正在运行的容器进程，得到其中的环境变量
	path := fmt.Sprintf("/proc/%s/environ", pid)
	exist, err := PathExist(path)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, fmt.Errorf("container process not exist")
	}

	contentBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("getEnvsByPid: read file %s failed, error: %v", path, err)
	}
	return strings.Split(string(contentBytes), "\u0000"), nil
}

func GetContainerInfo(dir fs.FileInfo) (*model.ContainerInfo, error) {
	// 目录名即为容器名
	containerName := dir.Name()
	configDir := fmt.Sprintf(model.DefaultInfoLocation, containerName)
	configFilePath := configDir + model.ConfigName

	content, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}

	info := new(model.ContainerInfo)
	err = json.Unmarshal(content, info)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func ImageIsExist(image string) (bool, error) {
	var imageName = image
	if !strings.Contains(image, "@") {
		imageName = fmt.Sprintf("%s@latest", image)
	}

	imageFullPath := fmt.Sprintf("%s%s.tar", model.DefaultImagePath, imageName)
	exist, err := PathExist(imageFullPath)
	if err != nil {
		return false, err
	}
	return exist, nil
}

func RandStringBytes(n int) string {
	letterBytes := "1234567890"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func DeleteContainerInfo(containerName string) {
	dirUrl := fmt.Sprintf(model.DefaultInfoLocation, containerName)
	err := os.RemoveAll(dirUrl)
	if err != nil {
		fmt.Println(fmt.Errorf("deleteContainerInfo: remove containerInfo Dir failed, error: %v", err))
	}
}

func FormatFileSize(size int64) string {
	var std int64 = 1024
	var formatSize string
	if size < std {  // 小于1KB
		formatSize = fmt.Sprintf("%.2fB", float64(size) / float64(1))
	} else if size < std * std {  // 小于1MB
		formatSize = fmt.Sprintf("%.2fKB", float64(size) / float64(std))
	} else if size < std * std * std {  // 小于1GB
		formatSize = fmt.Sprintf("%.2fMB", float64(size) / float64(std * std))
	} else if size < std * std * std * std {  // 小于1TB
		formatSize = fmt.Sprintf("%.2fGB", float64(size) / float64(std * std * std))
	} else if size < std * std * std * std * std {  // 小于1EB
		formatSize = fmt.Sprintf("%.2fTB", float64(size) / float64(std * std * std * std))
	} else {
		formatSize = fmt.Sprintf("%.2fEB", float64(size) / float64(std * std * std * std * std))
	}
	return formatSize
}

// IsSupportAufs 判断当前系统是否支持aufs联合文件系统
func IsSupportAufs() (bool, error) {
	ufsName := "aufs"
	// grep aufs /proc/filesystems
	cmd := exec.Command("grep", ufsName, "/proc/filesystems")

	// 创建一个用来获取命令执行输出的管道
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return false, err
	}

	// 执行命令
	err = cmd.Start()
	if err != nil {
		return false, err
	}

	// 获取命令执行的输出结果
	cmdResult, err := ioutil.ReadAll(outPipe)
	if err != nil {
		return false, err
	}

	exist := strings.Contains(string(cmdResult), ufsName)
	return exist, nil
}

// IsSupportOverlay 判断当前系统是否支持overlay联合文件系统
func IsSupportOverlay() (bool, error) {
	ufsName := "overlay"
	// grep overlay /proc/filesystems
	cmd := exec.Command("grep", ufsName, "/proc/filesystems")

	// 创建一个用来获取命令执行输出的管道
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return false, err
	}

	// 执行命令
	err = cmd.Start()
	if err != nil {
		return false, err
	}

	// 获取命令执行的输出结果
	cmdResult, err := ioutil.ReadAll(outPipe)
	if err != nil {
		return false, err
	}

	exist := strings.Contains(string(cmdResult), ufsName)
	return exist, nil
}

// CheckNetworkExists 检查是否存在指定的网络
func CheckNetworkExists(networkName string) (bool, error) {
	networkFile := fmt.Sprintf("%s%s", model.DefaultNetworkPath, networkName)
	exist, err := PathExist(networkFile)
	if err != nil {
		return false, err
	}

	return exist, nil
}

func RunCommand(cmdLine string) (string, error) {
	cmdArgs := strings.Split(cmdLine, " ")
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	fmt.Printf("Run Cmd: %s\n", cmd.String())
	// 使用缓冲区接收命令执行的输出和错误输出
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	// 执行命令
	err := cmd.Run()
	//fmt.Printf("out: %s\n", outBuf.String())
	//fmt.Printf("err: %s\n", errBuf.String())
	errOut := errBuf.String()
	if err != nil {
		return "", fmt.Errorf("RunCommand failed, error: %v, errOut: %s", err, errOut)
	}
	if errOut != "" && errOut != "\n" {
		return "", fmt.Errorf("RunCommand success, but errOut: %s", errOut)
	}
	return outBuf.String(), nil
}