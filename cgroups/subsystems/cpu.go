package subsystems

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

type CPUAmountSubsystem struct {

}

func (c *CPUAmountSubsystem) Name() string {
	return "cpuset"
}

func (c *CPUAmountSubsystem) Set(cgroupPath string, res *ResourceConfig) error {
	subsystemCgroupPath, err := GetCgroupPath(c.Name(), cgroupPath, true)
	if err != nil {
		return err
	}

	if res.CPUAmount != "" {
		targetFilePath := path.Join(subsystemCgroupPath, "cpuset.cpus")
		err = ioutil.WriteFile(targetFilePath, []byte(res.CPUAmount), 0644)
		if err != nil {
			return fmt.Errorf("set cgroup cpu amount failed, error: %v", err)
		}
	}
	return nil
}

func (c *CPUAmountSubsystem) AddProcess(cgroupPath string, pid int) error {
	subsystemCgroupPath, err := GetCgroupPath(c.Name(), cgroupPath, false)
	if err != nil {
		return err
	}

	targetFilePath2 := path.Join(subsystemCgroupPath, "cpuset.mems")
	err = ioutil.WriteFile(targetFilePath2, []byte("0"), 0644)
	if err != nil {
		return fmt.Errorf("cgroup add process failed, write cpuset.mems fail, error: %v", err)
	}

	targetFilePath := path.Join(subsystemCgroupPath, "tasks")
	err = ioutil.WriteFile(targetFilePath, []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		return fmt.Errorf("cgroup add process failed, write tasks fail, error: %v", err)
	}
	return nil
}

func (c *CPUAmountSubsystem) RemoveCgroup(cgroupPath string) error {
	subsystemCgroupPath, err := GetCgroupPath(c.Name(), cgroupPath, false)
	if err != nil {
		return err
	}

	return os.RemoveAll(subsystemCgroupPath)
}
