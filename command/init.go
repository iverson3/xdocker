package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

//这里的是 InitProcess，也就是容器初始化的步骤。
//注意 syscall.Exec 这句话。
// 1.注意书本上有个坑，就是没有 mount / 并指定 private。不然容器里的 proc 会使用外面的 proc。即使在不同的 namespace 下。
// 2.所以如果你没有加这一段，其实退出容器后你会发现你需要在外面再次 mount proc 才能使用 ps 等命令。

//一般来说，我们都是想要这个 containerCmd 作为 PID=1 的进程。但是很可惜，由于我们有 initProcess 半身的存在，所以 PID 为 1 的其实是 initProcess。那么如何让 containerCmd 作为 PID=1 的存在呢？
//这里就出现了 syscall.Exec 这个黑魔法，实际上 Exec 内部会调用 kernel 的 execve 函数，这个函数会把当前进程上运行的程序替换成另外一个程序，而这正是我们想要的，不改变 PID 的情况下，替换掉程序。（即使删除 PID 为 1 的进程，新创建的进程也会是 PID=2，所以必须要靠这个方法）

//为什么需要第一个命令的 PID 为 1？
//因为这样，退出这个进程后，容器就会因为没有前台进程，而自动退出。这也是 docker 的特性。

func InitChildProcess() (err error) {
	//defer func() {
	//	err2 := recover()
	//	if err2 != nil || err != nil {
	//		// 容器进程启动失败或程序panic，清理资源
	//		// 从环境变量中获取容器名和容器ID
	//		containerId := os.Getenv("xdocker_container_id")
	//		containerName := os.Getenv("xdocker_container_name")
	//		volume := os.Getenv("xdocker_volume")
	//
	//		rootUrl, err3 := pkg.GetContainerRootPath(containerId)
	//		if err3 != nil {
	//			return
	//		}
	//		mntUrl := rootUrl + "mnt/"
	//
	//		cGroupPath := fmt.Sprintf(model.DefaultCgroupPath, containerId)
	//		cm := cgroups.NewCgroupManager(cGroupPath)
	//
	//		// 在容器退出之前删除设置的aufs工作目录
	//		DeleteWorkSpace(rootUrl, mntUrl, volume)
	//		// 容器退出时，删除容器信息文件
	//		pkg.DeleteContainerInfo(containerName)
	//		// 删除限制资源的子系统相关目录
	//		_ = cm.Destroy()
	//	}
	//}()
	// 挂载相关设置
	err = setUpMount()
	if err != nil {
		return err
	}

	containerCmd := readCommand()
	if containerCmd == nil || len(containerCmd) == 0 {
		return fmt.Errorf("init process failed, containerCmd is nil")
	}

	//value, _ := syscall.Getenv("PATH")
	//fmt.Println("$PATH: ", value)

	// 设置系统PATH：新增 /bin 目录到PATH中，避免LookPath找不到命令
	path := os.Getenv("PATH")
	path = fmt.Sprintf("%s:/bin", path)
	err = os.Setenv("PATH", path)
	if err != nil {
		return err
	}

	// 这里我们添加了 lookPath，这个是用于解决每次我们都要输入 /bin/ls 的麻烦的，
	// 这个函数会帮我们找到参数命令的绝对路径。也就是说，你只需要输入 ls 即可，lookPath 会自动找到 /bin/ls 的。
	// 然后我们再把这个 path 作为 argv0 传给 syscall.Exec。
	cmdPath, err := exec.LookPath(containerCmd[0])
	if err != nil {
		fmt.Println(fmt.Errorf("ERROR: initProcess exec.LookPath failed, error: %v", err))
		return err
	}

	//go func() {
	//	ch := make(chan os.Signal)
	//	// 监听指定的信号
	//	signal.Notify(ch, os.Kill, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGABRT)
	//
	//	// 阻塞直到有信号到来 或者 退出通知的到来
	//	select {
	//	case <-ch:
	//		// 停止监听信号
	//		signal.Stop(ch)
	//
	//		// todo: 目前是监听不到宿主机的kill信号的
	//		fmt.Println("==============================")
	//		fmt.Println("got kill signal")
	//		fmt.Println("==============================")
	//
	//		// remove cgroup path
	//		//RemoveCgroupDirectory(cm)
	//		// 在容器退出之前删除设置的aufs工作目录
	//		//deleteWorkSpaceFunc()
	//	}
	//}()

	// 在容器内执行一下source，确保ENV设置的环境变量能生效
	//_, err = util.RunCommand(`sh -c source /etc/bashrc`)
	//if err != nil {
	//	// 只记录错误不影响后续的流程
	//	fmt.Println(fmt.Errorf("ERROR: initProcess source /etc/bashrc failed, error: %v", err))
	//}

	// 运行用户指定的命令或程序
	err = syscall.Exec(cmdPath, containerCmd, os.Environ())
	if err != nil {
		fmt.Println(fmt.Errorf("ERROR: exec '%s' failed, cmdArgv: %v, error: %v", cmdPath, containerCmd, err))
		return err
	}

	return nil
}

// 初始化挂载点
func setUpMount() error {
	// 首先设置根目录为私有模式，防止影响pivot_root
	// private方式挂载，不影响宿主机的挂载
	// 意思其实就是mount的传播问题：必须让父进程、子进程都不是分享模式。
	// pivot root 不允许 parent mount point 和 new mount point 是 shared。因为相互之间会进行传播影响。
	err := syscall.Mount("/", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("setUpMount: mount / failed, error: %v\n", err)
	}

	// 获取当前路径
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("setUpMount: get current location failed, error: %v", err)
	}

	err = privotRoot(pwd)
	if err != nil {
		return err
	}

	// 挂载proc文件系统
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	err = syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	if err != nil {
		return fmt.Errorf("setUpMount: mount /proc failed, error: %v\n", err)
	}

	// 挂载tmpfs文件系统
	//err = syscall.Mount("tmpfs", "/dev", "tempfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")
	//if err != nil {
	//	return fmt.Errorf("setUpMount: mount tmpfs failed, error: %v", err)
	//}
	return nil
}

// 对于pivot_root系统调用的使用还有一些约束条件：
// 主要约束条件：
// 1、new_root和put_old都必须是目录
// 2、new_root和put_old不能与当前根目录在同一个挂载上。
// 3、put_old必须是new_root，或者是new_root的子目录
// 4、new_root必须是一个挂载点，但不能是"/"。还不是挂载点的路径可以通过绑定将路径挂载到自身上转换为挂载点。
func privotRoot(root string) error {
	// 为了使当前root的老root和新root不在同一个文件系统下，把root重新mount一次
	// bind mount 是把相同的内容换了一个挂载点的挂载方式
	err := syscall.Mount(root, root, "bind", syscall.MS_BIND|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("privotRoot: mount rootfs to itself failed, error: %v", err)
	}

	pivotName := ".pivot_root"

	// 创建 rootfs/.pivot_root 存储old_root
	pivotDir := filepath.Join(root, pivotName)
	// 判断是否已存在该目录
	if _, err = os.Stat(pivotDir); err == nil {
		// 存在则删除
		if err = os.Remove(pivotDir); err != nil {
			return err
		}
	}

	if err = os.MkdirAll(pivotDir, 0777); err != nil {
		return fmt.Errorf("privotRoot: mkdir of pivot_root failed, error: %v", err)
	}

	// 当我们fork新的进程，子进程会使用父进程的文件系统。
	// 但如果我们想要把子进程的rootfs文件系统(/)修改成自定义的目录该怎么办呢？
	// 这时候就要使用 pivot_root系统调用了
	// 它的作用是将子进程的 / 更改为 new_root,原 / 存放到 put_old 文件夹下。
	// 挂载点目前依然可以在mount命令中看到
	// pivot_root改变当前进程所在mount namespace内的所有进程的root mount移到put_old，然后将new_root作为新的root mount；
	// root mount可以理解为rootfs，也就是“/”，pivot_root将所在mount namespace中的所有进程的“/”改为了new_root
	if err = syscall.PivotRoot(root, pivotDir); err != nil {
		return fmt.Errorf("privotRoot: syscall.PivotRoot() failed, error: %v", err)
	}

	// pivot_root并没有修改当前调用进程的工作目录，通常需要使用chdir(“/”)来实现切换到新的root mount的根目录
	// 修改当前工作目录到根目录
	if err = syscall.Chdir("/"); err != nil {
		return fmt.Errorf("privotRoot: chdir root failed, error: %v", err)
	}

	// 取消临时文件 .pivot_root 的挂载并删除它
	// 注意当前已经在根目录下，所以临时文件的目录也改变了
	pivotDir = filepath.Join("/", pivotName)
	if err = syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("privotRoot: unmount oivot_root dir failed, error: %v", err)
	}

	return os.Remove(pivotDir)
}

func readCommand() []string {
	// 对于标准输入、输出、错误,在创建子进程的时候都是默认带着的/继承的, 所以前三个文件描述符就是这三个
	// 第四个(下标3)则是我们的传过来的用来传递命令参数的管道
	pipe := os.NewFile(uintptr(3), "pipe")
	// 实际运行中，当进程运行到 readCommand() 的时候会堵塞，直到 write 端传数据进来。
	// 不用担心我们在容器运行后再传输参数。因为在读取完参数之前，init 函数也不会运行到 syscall.Exec 这一步
	msg, err := ioutil.ReadAll(pipe)
	if err != nil {
		fmt.Println(fmt.Errorf("ERROR: read pipe failed, error: %v\n", err))
		return nil
	}
	return strings.Split(string(msg), " ")
}