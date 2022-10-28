package subsystems

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
)

// 获取某个 subsystem 所挂载的 hieararchy 上的虚拟文件系统（挂载后的文件夹）下的 cgroup 的路径。
// 通过对这个目录的改写来改动 cgroup。
func GetCgroupPath(subsystemName string, cgroupPath string, autoCreate bool) (string, error) {
	cgroupRootPath := FindHierarchyMountRootPath(subsystemName)
	// 拼接得到绝对路径
	expectedPath := path.Join(cgroupRootPath, cgroupPath)

	if _, err := os.Stat(expectedPath); err == nil || (autoCreate && os.IsNotExist(err)) {
		if os.IsNotExist(err) {
			// 创建容器自己的子目录
			// 在 hierarchy 环境下，mkdir 其实会隐式地创建一个 cgroup，其中包含很多配置文件
			err = os.MkdirAll(expectedPath, os.ModePerm)
			if err != nil {
				return "", fmt.Errorf("create cgroup path failed, error: %v", err)
			}
		}
		return expectedPath, nil
	} else {
		return "", fmt.Errorf("cgroup path error: %v", err)
	}
}

// 找到 cgroup 的根节点
func FindHierarchyMountRootPath(subsystemName string) string {
	// 通过/proc/self/mountinfo文件里的内容找到对应子系统(cpu、memory、cpuset等)的cgroup root路径
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Text()
		fields := strings.Split(txt, " ")

		for _, opt := range strings.Split(fields[len(fields)-1], ","){
			if opt == subsystemName {
				return fields[4]
			}
		}
	}
	return ""
}
