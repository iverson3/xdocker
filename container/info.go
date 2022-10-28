package container

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"studygolang/docker/xdocker/model"
	"studygolang/docker/xdocker/util"
	"time"
)


func RecordContainerInfo(pid int, cmdArr []string, id, containerName, imageName, volume, networkName, ipAddress string, portMapping []string) error {
	createTime := time.Now().Format("2006-01-02 15:04:05")
	containerCmd := strings.Join(cmdArr, " ")

	containerInfo := &model.ContainerInfo{
		Pid:        strconv.Itoa(pid),
		ID:         id,
		Name:       containerName,
		Image:      imageName,
		Command:    containerCmd,
		Volume:     volume,
		CreateTime: createTime,
		Status:     model.RUNNING,
		NetworkName: networkName,
		IpAddress: ipAddress,
		PortMapping: portMapping,
	}

	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		return fmt.Errorf("recordContainerInfo: container info to json string failed, error: %v", err)
	}

	dirUrl := fmt.Sprintf(model.DefaultInfoLocation, containerName)
	if err = os.MkdirAll(dirUrl, 0666); err != nil {
		return fmt.Errorf("recordContainerInfo: mkdir %s failed, error: %v", dirUrl, err)
	}

	fileName := dirUrl + model.ConfigName
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("recordContainerInfo: create containerInfo file %s failed, error: %v", fileName, err)
	}
	defer file.Close()

	_, err = file.WriteString(string(jsonBytes))
	if err != nil {
		return fmt.Errorf("recordContainerInfo: containerInfo write into file failed, error: %v", err)
	}

	return nil
}

func RemoveInfoPath(containerName string) error {
	dirUrl := fmt.Sprintf(model.DefaultInfoLocation, containerName)
	exist, err := util.PathExist(dirUrl)
	if err != nil {
		return err
	}
	// 目录存在则删除
	if exist {
		err = os.RemoveAll(dirUrl)
		if err != nil {
			return err
		}
	}
	return nil
}

func CreateLogFile(containerName string) (*os.File, error) {
	dirUrl := fmt.Sprintf(model.DefaultInfoLocation, containerName)
	err := os.MkdirAll(dirUrl, 0644)   // 0777
	if err != nil {
		return nil, err
	}

	logPath := dirUrl + model.ContainerLogFileName
	f, err := os.Create(logPath)
	if err != nil {
		return nil, err
	}

	return f, nil
}