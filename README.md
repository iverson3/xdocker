# xdocker

#### **xdocker**是一个由Golang实现 仿Docker的应用容器引擎

> xdocker可基于镜像运行容器 (支持限制系统资源、挂载数据卷、端口映射、设置网络、设置环境变量)
>
> xdocker可基于Dockerfile多阶段构建镜像
>
> xdocker可进入运行中的容器进行操作
>
> xdocker可管理本地的镜像和容器
>
> xdocker可与远程的镜像仓库服务进行简单的交互  (自己实现的一个简单的镜像仓库服务)
>
> xdocker可管理网络，为容器的联网提供支持  (目前只支持bridge网络)



#### 已支持的命令列表：

- run      运行容器
- ps      列出容器
- inspect      获取容器的详细信息
- logs      输出容器的日志
- exec      进入容器
- pause      暂停容器
- continue      恢复容器
- start      启动一个已停止的容器
- stop      停止一个运行中的容器
- restart   重启一个运行中的容器
- rm        移除一个已停止的容器
- build      基于Dockerfile构建镜像 
- images      列出本地所有的镜像
- rmi      删除镜像  
- commit      基于容器创建一个新的镜像
- export      将容器打包为tar并导出
- list      列出远程镜像仓库中的镜像
- search      搜索远程镜像仓库中的镜像
- pull      从远程镜像仓库中拉取镜像
- push      将本地镜像推到远程镜像仓库中
- network list      列出网络
- network create      创建网络
- network remove      删除网络



#### 主要命令示例：

> xdocker run -it -name xxx  -cpuper 20 -m 100m -e GO111MODULE=on busybox sh    运行容器
>
> xdocker run -d -name xxx -v path1:path2 -net xdocker0 -p 8000:80 alpine gotcpserver     运行容器
>
> xdocker network create --driver bridge --subnet 192.168.10.1/24 xdocker0     创建网络
>
> xdocker build -t imagename@latest .    构建镜像
>
> xdocker exec 容器ID/容器名 sh     进入容器



#### xdocker系统相关目录：

> /usr/xdocker/images/                       镜像存储目录
>
> /usr/xdocker/containers/{容器ID}/   容器目录  (包含 容器只读层、容器读写层、容器rootfs目录)
>
> /usr/xdocker/info/{容器名}/              容器状态信息和日志文件的存储路径 
>
> /usr/xdocker/metadata/                   容器和镜像相关元数据 (比如 容器ID与容器名的映射关系)
>
> /usr/xdocker/network/                     容器网络信息存放目录     
>
> /usr/xdocker/volumes/                    数据卷目录



#### 如何运行xdocker

```
前提：  
1. 确保golang环境正常
2. 设置GO111MODULE确保开启gomod   go env -w GO111MODULE="on"
3. 设置GOPROXY确保依赖能够正常安装  go env -w GOPROXY=https://goproxy.cn,direct

make build                 在项目根目录下进行make
cp xdocker /usr/bin/       将可执行文件拷贝到某个PATH目录下 (可略过)
sudo xdocker images        确保以root权限去执行生成的xdocker可执行文件 (或者直接在root用户下操作)
```



#### Dockerfile已支持的命令列表：

- FROM
- RUN
- COPY
- WORKDIR
- ENV
- ARG
- ENTRYPOINT

*具体示例可查看Dockerfile文件*



#### 系统支持情况：

目前只支持ubuntu和centos，不支持windows

因为xdocker使用到的联合文件系统部分只实现了对 **aufs** 和 **overlay** 的支持，所以目前只能在支持aufs或overlay的系统上使用