package model

const (
	// DefaultNetworkName 默认的网络名
	DefaultNetworkName = "xdocker0"
	// DefaultNetworkDriver 默认的网络驱动
	DefaultNetworkDriver = "bridge"
	// DefaultNetworkSubnet 默认的网络子网
	DefaultNetworkSubnet = "192.168.10.1/24"

	// 容器的状态
	RUNNING = "running"
	PAUSED = "paused"
	STOP = "stop"
	EXIT = "exited"

	// DefaultInfoLocation 容器信息文件存放的默认路径 （其中 %s 代指具体的容器名）
	DefaultInfoLocation = "/usr/xdocker/info/%s/"
	// DefaultMetaDataLocation metadata相关元数据的存放目录
	DefaultMetaDataLocation = "/usr/xdocker/metadata/"
	// DefaultContainerRoot 容器的根目录 (其中 %s 表示具体的容器ID)
	DefaultContainerRoot = "/usr/xdocker/containers/%s/"
	// DefaultImagePath 镜像存储路径
	DefaultImagePath = "/usr/xdocker/images/"
	// DefaultCgroupPath cgroup路径(非完整路径，前面还有cgroup不同子系统的根路径)  (其中 %s 表示具体的容器ID)
	DefaultCgroupPath = "xdocker/%s"
	// DefaultNetworkPath 网络相关配置信息目录
	DefaultNetworkPath = "/usr/xdocker/network/network/"
	// IpamDefaultAllocatorPath ip分配管理器默认的网络信息的存储路径
	IpamDefaultAllocatorPath = "/usr/xdocker/network/ipam/subnet.json"
	// ConfigName 容器信息存储的文件名
	ConfigName = "config.json"
	// ContainerLogFileName 日志文件名
	ContainerLogFileName = "container.log"

	// DefaultImageHubServerUrl 默认的镜像仓库服务域名
	DefaultImageHubServerUrl = "http://81.69.56.251:8888"
	PushUrl = "/images/push"
	PullUrl = "/images/pull"
	ListUrl = "/images/list"
	SearchUrl = "/images/search"
)

// ContainerInfo 容器信息
type ContainerInfo struct {
	Pid string `json:"pid"`          // 容器的init进程在宿主机上的PID
	ID string `json:"id"`            // 容器ID
	Name string `json:"name"`        // 容器名
	Image string `json:"image"`      // 镜像名
	Command string `json:"command"`  // 容器运行命令
	Volume string `json:"volume"`    // 数据卷
	CreateTime string `json:"createTime"`
	Status string `json:"status"`    // 容器状态
	NetworkName string `json:"network_name"`  // 网络名
	IpAddress string `json:"ip_address"`      // 为容器分配的ip地址
	PortMapping []string `json:"port_mapping"`// 端口映射
}

// ImageInfo 镜像信息
type ImageInfo struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Size string `json:"size"`
	TAG string `json:"tag"`    // 版本
	CreateTime string `json:"createTime"`
}
