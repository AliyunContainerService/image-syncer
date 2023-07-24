# image-syncer

![workflow check](https://github.com/AliyunContainerService/image-syncer/actions/workflows/check.yml/badge.svg)
![workflow build](https://github.com/AliyunContainerService/image-syncer/actions/workflows/synctest.yml/badge.svg)
[![Version](https://img.shields.io/github/v/release/AliyunContainerService/image-syncer)](https://github.com/AliyunContainerService/image-syncer/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/AliyunContainerService/image-syncer)](https://goreportcard.com/report/github.com/AliyunContainerService/image-syncer)
[![Github All Releases](https://img.shields.io/github/downloads/AliyunContainerService/image-syncer/total.svg)](https://api.github.com/repos/AliyunContainerService/image-syncer/releases)
[![codecov](https://codecov.io/gh/AliyunContainerService/image-syncer/graph/badge.svg)](https://codecov.io/gh/AliyunContainerService/image-syncer)
[![License](https://img.shields.io/github/license/AliyunContainerService/image-syncer)](https://www.apache.org/licenses/LICENSE-2.0.html)

`image-syncer` 是一个容器镜像同步工具，可用来进行多对多的镜像仓库同步，支持目前绝大多数主流的 docker 镜像仓库服务

[English](./README.md) | 简体中文

## Features

- 支持多对多镜像仓库同步
- 支持基于 Docker Registry V2 搭建的镜像仓库服务 (如 Docker Hub、 Quay、 阿里云镜像服务 ACR、 Harbor 等)
- 同步过程只经过内存和网络，不依赖磁盘存储，同步速度快
- 自动增量同步, 自动忽略已同步且不需要修改的镜像
- 支持镜像层级别的并发同步，可以通过配置文件调整并发数（可以理解为同一时间在同步的镜像层数量上限）
- 自动重试失败的同步任务，可以解决大部分镜像同步中的偶发问题（限流、网络抖动），支持重试次数配置
- 简单轻量，不依赖 docker 以及其他程序

## 使用

### GitHub Action

可以使用 [image-sync-action](https://github.com/marketplace/actions/image-sync-action) 这个 github action 来实现灵活触发的镜像同步作业（比如定时同步），不需要支付任何资源的同时，也可以解决国内外网访问的问题

### 下载和安装

在 [releases](https://github.com/AliyunContainerService/image-syncer/releases) 页面可下载源码以及二进制文件

### 手动编译

```bash
go get github.com/AliyunContainerService/image-syncer
cd $GOPATH/github.com/AliyunContainerService/image-syncer

# This will create a binary file named image-syncer
make
```

### 命令用例

```bash
# 获得帮助信息
./image-syncer -h

./image-syncer --proc=6 --auth=./auth.json --images=./images.json --auth=./auth.json --retries=3
```

### 配置文件

为了提高配置的灵活性，image-syncer 支持通过 `--auth` 参数以文件的形式传入认证信息，同时通过 `--images` 参数以文件的形式传入镜像同步规则。两种配置文件都同时支持 YAML 和 JSON 两种格式，其中认证信息是可选的，镜像同步规则是必须的。通过两者分离的方式，可以做到认证信息的灵活复用

> 1.2.0 版本之前主要使用的、通过 `--config` 参数以一个配置文件同时传入认证信息和镜像同步规则的格式也是兼容的，可以参考 [config.yaml](examples/config.yaml) 和 [config.json](examples/config.json)

#### 认证信息

认证信息中可以同时描述多个 registry（或者 registry/namespace）对象，一个对象可以包含账号和密码，其中，密码可能是一个 TOKEN

> 注意，通常镜像源仓库需要具有 pull 以及访问 tags 权限，镜像目标仓库需要拥有 push 以及创建仓库权限；如果对应仓库没有提供认证信息，则默认匿名访问

认证信息文件通过 `--auth` 参数传入，具体文件样例可以参考 [auth.yaml](examples/auth.yaml) 和 [auth.json](examples/auth.json)，这里以 [auth.yaml](examples/auth.yaml) 为例：

```yaml
quay.io: #支持 "registry" 和 "registry/namespace"（v1.0.3之后的版本） 的形式，image-syncer 会自动为镜像同步规则中的每个源/目标 url 查找认证信息，并且使用对应认证信息进行进行访问，如果匹配到了多个，用“最长匹配”的那个作为最终结果
  username: xxx
  password: xxxxxxxxx
  insecure: true # 可选，（v1.0.1 之后支持）registry是否是http服务，如果是，insecure 字段需要为 true，默认是 false
registry.cn-beijing.aliyuncs.com:
  username: xxx # 可选，（v1.3.1 之后支持）value 使用 "${env}" 或者 "$env" 形式可以引用环境变量
  password: xxxxxxxxx # 可选，（v1.3.1 之后支持）value 使用 "${env}" 或者 "$env" 类型的字符串可以引用环境变量
docker.io:
  username: "${env}"
  password: "$env"
quay.io/coreos:
  username: abc
  password: xxxxxxxxx
  insecure: true
```

#### 镜像同步规则

每条镜像同步规则为一个 “源镜像 url: 目标镜像 url” 的键值对。无论是源镜像 url 还是目标镜像 url，字符串格式都和 docker pull 命令所使用的镜像 url 大致相同（registry/repository:tag、registry/repository@digest），但在 tag 和 digest 配置上和 docker pull 所使用的 url 存在区别，这里对整体逻辑进行描述：

1. 源镜像 url、目标镜像 url 都不能为空
2. 源镜像 url 不包含 tag 和 digest 时，代表同步源镜像 repository 中的所有镜像 tag
3. 源镜像 url 可以包含一个或多个 tag，多个 tag 之间用英文逗号分隔，代表同步源镜像 repository 中的多个指定镜像 tag
4. 源镜像 url 可以但最多只能包含一个 digest，此时如果目标镜像 url 包含 digest，digest 必须一致
5. 源镜像 url 的 "tag" 可以是一个正则表达式，需要额外在首尾加上 `/` 字符作为标识，源镜像 repository 中所有匹配正则表达式的镜像 tag 会被同步，不支持多个正则表达式
6. 目标镜像 url 可以不包含 tag 和 digest，表示所有需同步的镜像保持其镜像 tag 或者 digest 不变
7. 目标镜像 url 可以包含多个 tag 或者 digest，数量必须与源镜像 url 中的 tag 数量相同，此时，同步后的镜像 tag 会被修改成目标镜像 url 中指定的镜像 tag（按照从左到右顺序对应）
8. 支持同时指定多个目标镜像 url，此时 "目标镜像 url" 为数组的形式，数组的每个元素（字符串）都需要满足前面的规则

镜像同步规则文件通过 `--images` 参数传入，具体文件样例可以参考 [images.yaml](examples/images.yaml) 和 [images.json](examples/images.json)，这里以 [images.yaml](examples/images.yaml) 为例。 示例如下：

```yaml
quay.io/coreos/kube-rbac-proxy: quay.io/ruohe/kube-rbac-proxy
quay.io/coreos/kube-rbac-proxy:v1.0: quay.io/ruohe/kube-rbac-proxy
quay.io/coreos/kube-rbac-proxy:v1.0,v2.0: quay.io/ruohe/kube-rbac-proxy
quay.io/coreos/kube-rbac-proxy@sha256:14b267eb38aa85fd12d0e168fffa2d8a6187ac53a14a0212b0d4fce8d729598c: quay.io/ruohe/kube-rbac-proxy
quay.io/coreos/kube-rbac-proxy:v1.1:
  - quay.io/ruohe/kube-rbac-proxy1
  - quay.io/ruohe/kube-rbac-proxy2
quay.io/coreos/kube-rbac-proxy:/a+/: quay.io/ruohe/kube-rbac-proxy
```

### 更多参数

`image-syncer` 的使用比较简单，但同时也支持多个命令行参数的指定：

```
-h  --help       使用说明，会打印出一些启动参数的当前默认值

    --config     设置用户提供的配置文件路径，使用之前需要创建此文件，默认为当前工作目录下的config.json文件。这个参数与 --auth和--images 的
                 作用相同，分解成两个参数可以更好地区分认证信息与镜像仓库同步规则。建议使用 --auth 和 --images.

    --auth       设置用户提供的认证文件所在路径，使用之前需要创建此认证文件，默认为当前工作目录下的auth.json文件

    --images     设置用户提供的镜像同步规则文件所在路径，使用之前需要创建此文件，默认为当前工作目录下的images.json文件

    --log        打印出来的log文件路径，默认打印到标准错误输出，如果将日志打印到文件将不会有命令行输出，此时需要通过cat对应的日志文件查看

    --proc       并发数，进行镜像同步的并发goroutine数量，默认为5

    --retries    失败同步任务的重试次数，默认为2，重试会在所有任务都被执行一遍之后开始，并且也会重新尝试对应次数生成失败任务的生成。一些偶尔出现的网络错误比如io timeout、TLS handshake timeout，都可以通过设置重试次数来减少失败的任务数量

    --os         用来过滤源 tag 的 os 列表，为空则没有任何过滤要求，只对非 docker v2 schema1 media 类型的镜像格式有效

    --arch       用来过滤源 tag 的 architecture 列表，为空则没有任何过滤要求

    --force      同步已经存在的、被忽略的镜像，这个操作会更新已存在镜像的时间戳
```

### FAQs

同步中常见的问题汇总在[FAQs 文档](./FAQs.md)中

### Star History

[![Star History Chart](https://api.star-history.com/svg?repos=AliyunContainerService/image-syncer&type=Date)](https://star-history.com/#AliyunContainerService/image-syncer)
