package subsystems

type CPUShareSubsystem struct {

}

func (c *CPUShareSubsystem) Name() string {
	return "cpu"
}

func (c *CPUShareSubsystem) Set(cgroupPath string, res *ResourceConfig) error {
	return nil
}

func (c *CPUShareSubsystem) AddProcess(cgroupPath string, pid int) error {
	return nil
}

func (c *CPUShareSubsystem) RemoveCgroup(cgroupPath string) error {
	return nil
}