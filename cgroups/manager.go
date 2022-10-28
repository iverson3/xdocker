package cgroups

import (
	"fmt"
	"studygolang/docker/xdocker/cgroups/subsystems"
)

type CgroupManager struct {
	// 相对路径，相对的是对应的 hierarchy 的 root path
	// 所以一个 CgroupManagee 是有可能表示多个 cgroups 的，或者准确来说，和对应的 hierarchy root path 的相对路径一样的多个 cgroups。
	Path string
	Resource *subsystems.ResourceConfig
}

func NewCgroupManager(path string) *CgroupManager {
	return &CgroupManager{
		Path: path,
	}
}

// Set 设置子系统限制
// 可能会创建多个 cgroups，如果 subsystems 们在不同的 hierarchy 上的话就会这样
func (c *CgroupManager) Set(res *subsystems.ResourceConfig) error {
	for _, subsystem := range subsystems.SubsystemsInstance {
		err := subsystem.Set(c.Path, res)
		if err != nil {
			// 注意 set 和 addProcess 都不是返回错误，而是发出警告，然后返回 nil。
			// 因为有些时候用户只指定某一个限制，比如 memory。
			// 那样的话修改 cpu 等其实会报错的，这是正常的报错，因此我们不 return err 来退出
			fmt.Println(fmt.Errorf("set resource failed, error: %v", err))
		}
	}
	return nil
}

// AddProcess 将当前进程放入各个子系统的cgroup中
func (c *CgroupManager) AddProcess(pid int) error {
	//AddProcess 和 Remove 都要在每个 subsystem 上执行一遍。因为这些 subsystem 可能存在于不同的 hierarchies 上。
	for _, subsystem := range subsystems.SubsystemsInstance {
		err := subsystem.AddProcess(c.Path, pid)
		if err != nil {
			fmt.Println(fmt.Errorf("add process failed, error: %v", err))
		}
	}
	return nil
}

// Destroy 销毁各个子系统中的cgroup
func (c *CgroupManager) Destroy() error {
	// AddProcess 和 Remove 都要在每个 subsystem 上执行一遍。因为这些 subsystem 可能存在于不同的 hierarchies 上。
	for _, subsystem := range subsystems.SubsystemsInstance {
		err := subsystem.RemoveCgroup(c.Path)
		if err != nil {
			fmt.Println(fmt.Errorf("ERROR: remove cgroup dir failed, path: %s, subsystem-name: %s, error: %v", c.Path, subsystem.Name(), err))
			//return err
		}
	}
	return nil
}