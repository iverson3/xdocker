package subsystems

import "strings"

type ResourceConfig struct {
	MemoryLimit string
	CPUPercentage int
	CPUShare string
	CPUAmount string
}

type Subsystem interface {
	// 返回subsystem的类型名
	Name() string
	// 为指定的cgroup设置资源限制
	Set(cgroupPath string, res *ResourceConfig) error
	// 添加一个进程到指定的cgroup
	AddProcess(cgroupPath string, pid int) error
	// 移除一个cgroup
	RemoveCgroup(cgroupPath string) error
}

var SubsystemsInstance = []Subsystem{
	&CPUPercentageSubsystem{},
	&CPUShareSubsystem{},
	//&CPUAmountSubsystem{},
	&MemorySubsystem{},
}

func (r *ResourceConfig) String() string {
	var line []string
	line = append(line, "MemoryLimit:", r.MemoryLimit)
	line = append(line, "CpuShare:", r.CPUShare)
	line = append(line, "CpuSet:", r.CPUAmount)
	return strings.Join(line, " ")
}