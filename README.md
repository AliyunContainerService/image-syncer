# image-syncer

![workflow check](https://github.com/AliyunContainerService/image-syncer/actions/workflows/check.yml/badge.svg)
![workflow build](https://github.com/AliyunContainerService/image-syncer/actions/workflows/build.yml/badge.svg)
[![Version](https://img.shields.io/github/v/release/AliyunContainerService/image-syncer)](https://github.com/AliyunContainerService/image-syncer/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/AliyunContainerService/image-syncer)](https://goreportcard.com/report/github.com/AliyunContainerService/image-syncer)
[![Github All Releases](https://img.shields.io/github/downloads/AliyunContainerService/image-syncer/total.svg)](https://api.github.com/repos/AliyunContainerService/image-syncer/releases)
[![codecov](https://codecov.io/gh/AliyunContainerService/image-syncer/graph/badge.svg)](https://codecov.io/gh/AliyunContainerService/image-syncer)
[![License](https://img.shields.io/github/license/AliyunContainerService/image-syncer)](https://www.apache.org/licenses/LICENSE-2.0.html)

`image-syncer` is a docker registry tools. With `image-syncer` you can synchronize docker images from some source registries to target registries, which include most popular public docker registry services.

English | [简体中文](./README-zh_CN.md)

## Features

- Support for many-to-many registry synchronization
- Supports docker registry services based on Docker Registry V2 (e.g., Alibaba Cloud Container Registry Service, Docker Hub, Quay.io, Harbor, etc.)
- Network & Memory Only, don't rely on large disk storage, fast synchronization
- Incremental Synchronization, use a disk file to record the synchronized image blobs' information
- Concurrent Synchronization, adjustable goroutine numbers
- Automatic Retries of Failed Sync Tasks, to resolve the network problems while synchronizing
- Doesn't rely on Docker daemon or other programs

## Usage

### Install image-syncer

You can download latest binary release [here](https://github.com/AliyunContainerService/image-syncer/releases)

### Compile Manually

```
go get github.com/AliyunContainerService/image-syncer
cd $GOPATH/github.com/AliyunContainerService/image-syncer

# This will create a binary file named image-syncer
make
```

### Example

```shell
# Get usage information
./image-syncer -h

# With this command, configure file path is "./config.json", default target registry is "registry.cn-beijing.aliyuncs.com",
# default target namespace is "ruohe", 6 of goroutine numbers, every failed task will be retried 3 times.
./image-syncer --proc=6 --auth=./auth.json --images=./images.json --namespace=ruohe \
--registry=registry.cn-beijing.aliyuncs.com --retries=3
```

### Configure Files

After v1.2.0, image-syncer supports both YAML and JSON format, and origin config file can be split into "auth" and "images" file. A full list of examples can be found under [example](./example), meanwhile the older version of configuration file is still supported via --config flag.

#### Authentication file

Authentication file holds all the authentication information for each registry, the following is an example of `auth.json`

```java
{
    // Authentication fields, each object has a URL as key and a username/password pair as value,
    // if authentication object is not provided for a registry, access to the registry will be anonymous.

    "quay.io": [{        // This "registry" or "registry/namespace" string should be the same as registry or registry/namespace used below in "images" field.
                            // The format of "registry/namespace" will be more prior matched than "registry"
        "username": "xxx",       // Optional, if the value is a string of "${env}" or "$env", image-syncer will try to find the value in environment variables, after v1.3.1
        "password": "xxxxxxxxx", // Optional, if the value is a string of "${env}" or "$env", image-syncer will try to find the value in environment variables, after v1.3.1
        "insecure": true         // "insecure" field needs to be true if this registry is a http service, default value is false, version of image-syncer need to be later than v1.0.1 to support this field
    }],
    "registry.cn-beijing.aliyuncs.com": [{
        "username": "xxx",
        "password": "xxxxxxxxx"
    }],
    "registry.hub.docker.com": [{
        "username": "xxx",
        "password": "xxxxxxxxxx"
    }],
    "quay.io/coreos": [{     // "registry/namespace" format is supported after v1.0.3 of image-syncer
        "username": "abc",
        "password": "xxxxxxxxx",
        "insecure": true
    }]
}
```

#### Image sync configuration file

Image sync configuration file defines all the image synchronization rules, the following is an example of `images.json`

```java
{
    // Rules of image synchronization, each rule is a kv pair of source(key) and destination(value).

    // The source of each rule should not be empty string.

    // If you need to synchronize images from one source to multi destinations, add more rules.

    // Both source and destination are docker image url (registry/namespace/repository:tag),
    // with or without tags.

    // For both source and destination, if destination is not an empty string, "registry/namespace/repository"
    // is needed at least.

    // You cannot synchronize a whole namespace or a registry but a repository for one rule at most.

    // The repository name and tag of destination can be deferent from source, which works like
    // "docker pull + docker tag + docker push"

    "quay.io/coreos/kube-rbac-proxy": "quay.io/ruohe/kube-rbac-proxy",
    "xxxx":"xxxxx",
    "xxx/xxx/xx:tag1,tag2,tag3":"xxx/xxx/xx"

    // If a source doesn't include tags, it means all the tags of this repository need to be synchronized,
    // destination should not include tags at this moment.

    // Each source can include more than one tags, which is split by comma (e.g., "a/b/c:1", "a/b/c:1,2,3").

    // If a source includes just one tag (e.g., "a/b/c:1"), it means only one tag need to be synchronized;
    // at this moment, if the destination doesn't include a tag, synchronized image will keep the same tag.

    // When a source includes more than one tag (e.g., "a/b/c:1,2,3"), at this moment,
    // the destination should not include tags, synchronized images will keep the original tags.
    // e.g., "a/b/c:1,2,3":"x/y/z".

    // When a destination is an empty string, source will be synchronized to "default-registry/default-namespace"
    // with the same repository name and tags, default-registry and default-namespace can be set by both parameters
    // and environment variable.
}
```

### Parameters

```
-h  --help       usage information

    --config     set the path of config file, this file need to be created before starting synchronization, default
                 config file is at "current/working/directory/config.json". (This flag can be replaced with flag --auth
                 and --images which for better orgnization.)

    --auth       set the path of authentication file, this file need to be created before starting synchronization, default
                 config file is at "current/working/directory/auth.json". This flag need to be pair used with --images.

    --images     set the path of image rules file, this file need to be created before starting synchronization, default
                 config file is at "current/working/directory/images.json". This flag need to be pair used with --auth.

    --log        set the path of log file, logs will be printed to Stderr by default

    --namespace  set default-namespace, default-namespace can also be set by environment variable "DEFAULT_NAMESPACE",
                 if they are both set at the same time, "DEFAULT_NAMESPACE" will not work at this synchronization,
                 default-namespace will work only if default-registry is not empty.

    --registry   set default-registry, default-registry can also be set by environment variable "DEFAULT_REGISTRY",
                 if they are both set at the same time, "DEFAULT_REGISTRY" will not work at this synchronization,
                 default-registry will work only if default-namespace is not empty.

    --proc       number of goroutines, default value is 5

    --records    image-syncer will record the information of synchronized image blobs to a disk file, this parameter will
                 set the path of the records file, default path is "current/working/directory/records", a records file can be
                 reused to make incremental synchronization if it is really generated by yourself. image-syncer remove the
                 dependence of records file after v1.1.0

    --retries    number of retries, default value is 2, the retries of failed sync tasks will start after all sync tasks
                 are executed once, reties of failed sync tasks will resolve most occasional network problems during
                 synchronization

    --os         os list to filter source tags, not works for docker v2 schema1 media, takes no effect if empty

    --arch       architecture list to filter source tags, takes no effect if empty
```

### FAQs

Frequently asked questions are listed in [FAQs](./FAQs.md)
