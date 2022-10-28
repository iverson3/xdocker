package subsystems

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

type CPUPercentageSubsystem struct {

}

func (c *CPUPercentageSubsystem) Name() string {
	return "cpu"
}

func (c *CPUPercentageSubsystem) Set(cgroupPath string, res *ResourceConfig) error {
	subsystemCgroupPath, err := GetCgroupPath(c.Name(), cgroupPath, true)
	if err != nil {
		return err
	}

	targetFilePath := path.Join(subsystemCgroupPath, "cpu.cfs_quota_us")

	// 如果没传参数，默认设置为 -1
	// -1表示cpu使用无限制
	// CPUPercentage = 20 表示cpu使用上限为 20%
	// CPUPercentage = 200 表示cpu使用上限为 200% 即两个cpu核 (前提是至少两个cpu核)
	var percentage = -1
	if res.CPUPercentage != -1 && res.CPUPercentage != 0 {
		percentage = res.CPUPercentage * 1000
	}

	err = ioutil.WriteFile(targetFilePath, []byte(strconv.Itoa(percentage)), 0644)
	if err != nil {
		return fmt.Errorf("set cgroup cpu amount failed, error: %v", err)
	}
	return nil
}

func (c *CPUPercentageSubsystem) AddProcess(cgroupPath string, pid int) error {
	subsystemCgroupPath, err := GetCgroupPath(c.Name(), cgroupPath, false)
	if err != nil {
		return err
	}
	// todo: 去重  cpu_share 和 cpu_percentage 在同一个cgroup目录下，同一个进程会在tasks中添加两次
	targetFilePath := path.Join(subsystemCgroupPath, "tasks")
	err = ioutil.WriteFile(targetFilePath, []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		return fmt.Errorf("cgroup add process failed, write tasks fail, error: %v", err)
	}
	return nil
}

func (c *CPUPercentageSubsystem) RemoveCgroup(cgroupPath string) error {
	subsystemCgroupPath, err := GetCgroupPath(c.Name(), cgroupPath, false)
	if err != nil {
		return err
	}

	return os.RemoveAll(subsystemCgroupPath)
}
