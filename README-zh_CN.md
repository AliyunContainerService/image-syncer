# image-syncer

[![CircleCI](https://circleci.com/gh/AliyunContainerService/image-syncer.svg?style=svg)](https://circleci.com/gh/AliyunContainerService/image-syncer)
[![Go Report Card](https://goreportcard.com/badge/github.com/AliyunContainerService/image-syncer)](https://goreportcard.com/report/github.com/AliyunContainerService/image-syncer)
[![codecov](https://codecov.io/gh/AliyunContainerService/image-syncer/graph/badge.svg)](https://codecov.io/gh/AliyunContainerService/image-syncer)

`image-syncer` 是一个docker镜像同步工具，可用来进行多对多的镜像仓库同步，支持目前绝大多数主流的docker镜像仓库服务

[English](./README.md) | 简体中文

## Features

- 支持多对多镜像仓库同步
- 支持基于Docker Registry V2搭建的docker镜像仓库服务 (如 Docker Hub、 Quay、 阿里云镜像服务ACR、 Harbor等)
- 同步只经过内存和网络，不依赖磁盘存储，同步速度快
- 增量同步, 通过对同步过的镜像blob信息落盘，不重复同步已同步的镜像
- 并发同步，可以通过配置文件调整并发数
- 自动重试失败的同步任务，可以解决大部分镜像同步中的网络抖动问题
- 不依赖docker以及其他程序

## 使用

### 下载和安装

在[releases](https://github.com/AliyunContainerService/image-syncer/releases)页面可下载源码以及二进制文件

### 手动编译

```
go get github.com/AliyunContainerService/image-syncer
cd $GOPATH/github.com/AliyunContainerService/image-syncer

# This will create a binary file named image-syncer
make
```

### 使用用例

```shell
# 获得帮助信息
./image-syncer -h

# 设置配置文件为config.json，默认registry为registry.cn-beijing.aliyuncs.com
# 默认namespace为ruohe，并发数为6
./image-syncer --proc=6 --config=./config.json --namespace=ruohe \
--registry=registry.cn-beijing.aliyuncs.com --retries=3
```

<!-- 
### 同步镜像到ACR

ACR(Ali Container Registry) 是阿里云提供的容器镜像服务，ACR企业版(EE)提供了企业级的容器镜像、Helm Chart 安全托管能力，推荐安全需求高、业务多地域部署、拥有大规模集群节点的企业级客户使用。

这里会将quay.io上的一些镜像同步到ACR企业版，作为使用用例。

#### 创建企业版ACR

1. [创建容器镜像服务]()
2.  -->

### 配置文件

```java
{
    "auth": {                   // 认证字段，其中每个对象为一个registry的一个账号和
                                // 密码；通常，同步源需要具有pull以及访问tags权限，
                                // 同步目标需要拥有push以及创建仓库权限，如果没有提供，则默认匿名访问
        
        "quay.io": {            // 支持 "registry" 和 "registry/namespace"（v1.0.3之后的版本） 的形式，需要跟下面images中的registry(registry/namespace)对应
                                // images中被匹配到的的url会使用对应账号密码进行镜像同步, 优先匹配 "registry/namespace" 的形式
            "username": "xxx",               // 用户名，可选
            "password": "xxxxxxxxx",         // 密码，可选
            "insecure": true                 // registry是否是http服务，如果是，insecure 字段需要为true，默认是false，可选，支持这个选项需要image-syncer版本 > v1.0.1
        },
        "registry.cn-beijing.aliyuncs.com": {
            "username": "xxx",
            "password": "xxxxxxxxx"
        },
        "registry.hub.docker.com": {
            "username": "xxx",
            "password": "xxxxxxxxxx"
        },
        "quay.io/coreos": {                       
            "username": "abc",              
            "password": "xxxxxxxxx",
            "insecure": true  
        }
    },
    "images": {
        // 同步镜像规则字段，其中条规则包括一个源仓库（键）和一个目标仓库（值）
        
        // 同步的最大单位是仓库（repo），不支持通过一条规则同步整个namespace以及registry
        
        // 源仓库和目标仓库的格式与docker pull/push命令使用的镜像url类似（registry/namespace/repository:tag）
        // 源仓库和目标仓库（如果目标仓库不为空字符串）都至少包含registry/namespace/repository
        // 源仓库字段不能为空，如果需要将一个源仓库同步到多个目标仓库需要配置多条规则
        // 目标仓库名可以和源仓库名不同（tag也可以不同），此时同步功能类似于：docker pull + docker tag + docker push

        "quay.io/coreos/kube-rbac-proxy": "quay.io/ruohe/kube-rbac-proxy",
        "xxxx":"xxxxx",
        "xxx/xxx/xx:tag1,tag2,tag3":"xxx/xxx/xx"

        // 当源仓库字段中不包含tag时，表示将该仓库所有tag同步到目标仓库，此时目标仓库不能包含tag
        // 当源仓库字段中包含tag时，表示只同步源仓库中的一个tag到目标仓库，如果目标仓库中不包含tag，则默认使用源tag
        // 源仓库字段中的tag可以同时包含多个（比如"a/b/c:1,2,3"），tag之间通过","隔开，此时目标仓库不能包含tag，并且默认使用原来的tag
        
        // 当目标仓库为空字符串时，会将源镜像同步到默认registry的默认namespace下，并且repo以及tag与源仓库相同，默认registry和默认namespace可以通过命令行参数以及环境变量配置，参考下面的描述
    }	 
}	
     
```

### 更多参数

`image-syncer` 的使用比较简单，但同时也支持多个命令行参数的指定：

```
-h  --help       使用说明，会打印出一些启动参数的当前默认值

    --config     设置用户提供的配置文件所在路径，使用之前需要创建配置文件，默认为当前工作目录下的image-syncer.json文件

    --log        打印出来的log文件路径，默认打印到标准错误输出，如果将日志打印到文件将不会有命令行输出，此时需要通过cat对应的日志文件查看

    --namespace  设置默认的目标namespace，当配置文件内一条images规则的目标仓库为空，并且默认registry也不为空时有效，可以通过环境变量DEFAULT_NAMESPACE设置，同时传入命令行参数会优先使用命令行参数值

    --registry   设置默认的目标registry，当配置文件内一条images规则的目标仓库为空，并且默认namespace也不为空时有效，可以通过环境变量DEFAULT_REGISTRY设置，同时传入命令行参数会优先使用命令行参数值

    --proc       并发数，进行镜像同步的并发goroutine数量，默认为5

    --records    指定传输过程中保存已传输完成镜像信息（blob）的文件输出/读取路径，默认输出到当前工作目录，一个records记录了对应目标仓库的已迁移信息，可以用来进行连续的多次迁移（会节约大量时间，但不要把之前自己没执行过的records文件拿来用），如果有unknown blob之类的错误，可以删除该文件重新尝试

    --retries    失败同步任务的重试次数，默认为2，重试会在所有任务都被执行一遍之后开始，并且也会重新尝试对应次数生成失败任务的生成。一些偶尔出现的网络错误比如io timeout、TLS handshake timeout，都可以通过设置重试次数来减少失败的任务数量
```

### FAQs

同步中常见的问题汇总在[FAQs文档](./FAQs.md)中