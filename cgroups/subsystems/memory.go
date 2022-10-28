package subsystems

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

type MemorySubsystem struct {

}

func (m *MemorySubsystem) Name() string {
	return "memory"
}

func (m *MemorySubsystem) Set(cgroupPath string, res *ResourceConfig) error {
	subsystemCgroupPath, err := GetCgroupPath(m.Name(), cgroupPath, true)
	if err != nil {
		return err
	}

	// 如果用户没有设置内存限制，则使用默认的内存限制
	// 目前是写死一个数，后期可以获取系统的最大内存数或者从配置文件中获取
	memoryLimit := "512m"
	if res.MemoryLimit != "" {
		memoryLimit = res.MemoryLimit
	}

	// 将内存限制写入对应的文件中，即可达到限制资源的目的
	err = ioutil.WriteFile(path.Join(subsystemCgroupPath, "memory.limit_in_bytes"), []byte(memoryLimit), 0644)
	if err != nil {
		return fmt.Errorf("set cgroup memory limit failed, error: %v", err)
	}
	return nil
}

func (m *MemorySubsystem) AddProcess(cgroupPath string, pid int) error {
	subsystemCgroupPath, err := GetCgroupPath(m.Name(), cgroupPath, false)
	if err != nil {
		return err
	}

	// 将进程的pid写入对应的文件中，即完成了将进程添加到了指定的cgroup中
	err = ioutil.WriteFile(path.Join(subsystemCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		return fmt.Errorf("cgroup add process failed, error: %v", err)
	}
	return nil
}

func (m *MemorySubsystem) RemoveCgroup(cgroupPath string) error {
	subsystemCgroupPath, err := GetCgroupPath(m.Name(), cgroupPath, false)
	if err != nil {
		fmt.Println(fmt.Errorf("get cgroup path failed, error: %v", err))
		return err
	}

	// 使用 os.Remove 可以移除参数所指定路径的文件或者文件夹。
	// 这里移除整个 cgroup 文件夹，就等于是删除 cgroup
	return os.RemoveAll(subsystemCgroupPath)
}
