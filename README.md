# image-syncer

[![Go Report Card](https://goreportcard.com/badge/github.com/AliyunContainerService/image-syncer)](https://goreportcard.com/report/github.com/AliyunContainerService/image-syncer)

`image-syncer` is a docker registry tools. With `image-syncer` you can synchronize docker images from some source registries to target registries, which include most popular public docker registry services.

English | [简体中文](./README-zh_CN.md)

## Features

- Support docker registry services based on Docker Registry V2 (e.g., AliContainerRegistry, dockerhub, quay, harbor)
- Network & Memory Only, don't rely on large disk storage, fast synchronization
- Incremental Synchronization, use a disk file to record the synchronized image blobs' information
- Concurrent Synchronization, adjustable goroutine numbers
- Automatic Retries of Failed Sync Tasks, to resolve the network problems while synchronizing
- Don't rely on Docker Deamon or other programs

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
# Get usage infomation
./image-syncer -h

# With this command, configure file path is "./config.json", default target registry is "registry.cn-beijing.aliyuncs.com",
# default target namespace is "ruohe", 6 of goroutine numbers, every failed task will be retried 3 times.
./image-syncer --proc=6 --config=./config.json --namespace=ruohe \
--registry=registry.cn-beijing.aliyuncs.com --retries=3
```

### Configure File

```java
{
    "auth": {               // Authentication fields, each object has a URL as key and a username/password pair as value, 
                            // if authentication object is not provided for a registry, access to the registry will be anonymous.
        
        "quay.io": {        // This URL of registry should be the same as registry used below in "images fields".
            "username": "xxx",             
            "password": "xxxxxxxxx",
            "insecure": true         // "insecure" field needs to be true if this registry is a http service, default value is false, version of image-syncer need to be later than v1.0.1 to support this field
        },
        "registry.cn-beijing.aliyuncs.com": {
            "username": "xxx",
            "password": "xxxxxxxxx"
        },
        "registry.hub.docker.com": {
            "username": "xxx",
            "password": "xxxxxxxxxx"
        }
    },
    "images": {
        // Rules of image synchronization, each rule is a kv pair of source(key) and destination(value). 
        // The source of each rule should not be empty string.
        // If you need to synchronize images from one source to multi destinations, add more rules.

        // Both source and destination are docker repository url almostly (repository/namespace:tag), 
        // with or without tags.

        "quay.io/coreos/kube-rbac-proxy": "quay.io/ruohe/kube-rbac-proxy",
        "xxxx":"xxxxx",
        "xxx/xxx/xx:tag1,tag2,tag3":"xxx/xxx/xx"
        // If a source doesn't include tags, it means all the tags of this repository need to be synchronized,
        // destination should not include tags at this moment.
        
        // Each source can include more than one tags, which is split by comma (e.g., "a/b/c:1", "a/b/c:1,2,3").

        // If a source includes just one tag (e.g., "a/b/c:1"), it means only one tag need to be synchronized;
        // at this moment, if the destination doesn't include a tag, synchronized image will keep the same tag.
        
        // When a source includes more than one tag (e.g., "a/b/c:1,2,3"), at this moment,
        // the destination should not include tags, synchronized images will keep the same tag tags.
        // e.g., "a/b/c:1,2,3":"x/y/z".
        
        // When a destination is an empty string, source will be synchronized to "default-registry/default-namespace"
        // with the same repository name and tags, default-registry and default-namespace can be set by both parameters
        // and environment variable.
    }	 
}	
     
```

### Parameters

```
-h  --help       usage information

    --config     set the path of configure file, this file need to be created before starting synchronization, default
                 config file is at "current/working/directory/image-syncer.json"

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
                 reused to make incremental synchronization if it is really generated by yourself

    --retires    number of retries, default value is 2, the retries of failed sync tasks will start after all sync tasks
                 are executed once, reties of failed sync tasks will resolve most occasional network problems during 
                 synchronization
```


### FAQs

Frequently asked questions are listed in [FAQs](./FAQs.md)
