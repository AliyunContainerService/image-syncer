package client

import (
	"container/list"
	"fmt"
	"strings"

	"github.com/AliyunContainerService/image-syncer/pkg/sync"
	"github.com/AliyunContainerService/image-syncer/pkg/tools"
	"github.com/sirupsen/logrus"
)

type Client struct {
	// a SyncTask list
	taskList *list.List

	// a UrlPair list
	urlPairList *list.List

	// failed list
	failedTaskList         *list.List
	failedTaskGenerateList *list.List

	config *Config

	routineNum int
	retries    int
	logger     *logrus.Logger

	// mutex
	taskListChan               chan int
	urlPairListChan            chan int
	failedTaskListChan         chan int
	failedTaskGenerateListChan chan int
}

type UrlPair struct {
	source      string
	destination string
}

func NewSyncClient(configFile, logFile, recordsFile string, routineNum, retries int, defaultDestRegistry string, defaultDestNamespace string) (*Client, error) {
	logger := NewFileLogger(logFile)

	config, err := NewSyncConfig(configFile, defaultDestRegistry, defaultDestNamespace)
	if err != nil {
		return nil, fmt.Errorf("Generate confg error: %v", err)
	}

	// init blob recorder
	sync.NewSyncBlobRecorder(recordsFile)

	return &Client{
		taskList:                   list.New(),
		urlPairList:                list.New(),
		failedTaskList:             list.New(),
		failedTaskGenerateList:     list.New(),
		config:                     config,
		routineNum:                 routineNum,
		retries:                    retries,
		logger:                     logger,
		taskListChan:               make(chan int, 1),
		urlPairListChan:            make(chan int, 1),
		failedTaskListChan:         make(chan int, 1),
		failedTaskGenerateListChan: make(chan int, 1),
	}, nil
}

func (c *Client) Run() {
	fmt.Println("Start to generate sync tasks, please wait ...")

	var finishChan = make(chan int, c.routineNum)

	// opem num of goroutines and wait c for close
	openRoutinesGenTaskAndWaitForFinish := func() {
		for i := 0; i < c.routineNum; i++ {
			go func() {
				for {
					urlPair, empty := c.GetAUrlPair()
					// no more task to generate
					if empty {
						break
					}
					moreUrlPairs, err := c.GenerateSyncTask(urlPair.source, urlPair.destination)
					if err != nil {
						c.logger.Errorf("Generate sync task %s to %s error: %v", urlPair.source, urlPair.destination, err)
						// put to failedTaskGenerateList
						c.PutAFailedUrlPair(urlPair)
					}
					if moreUrlPairs != nil {
						c.PutUrlPairs(moreUrlPairs)
					}
				}
				finishChan <- 1
			}()
		}
		for i := 0; i < c.routineNum; i++ {
			<-finishChan
		}
	}

	openRoutinesHandleTaskAndWaitForFinish := func() {
		for i := 0; i < c.routineNum; i++ {
			go func() {
				for {
					task, empty := c.GetATask()
					// no more tasks need to handle
					if empty {
						break
					}
					if err := task.Run(); err != nil {
						// put to failedTaskList
						c.PutAFailedTask(task)
					}
				}
				finishChan <- 1
			}()
		}

		for i := 0; i < c.routineNum; i++ {
			<-finishChan
		}
	}

	for source, dest := range c.config.GetImageList() {
		c.urlPairList.PushBack(&UrlPair{
			source:      source,
			destination: dest,
		})
	}

	// generate sync tasks
	openRoutinesGenTaskAndWaitForFinish()

	fmt.Println("Start to handle sync tasks, please wait ...")

	// generate goroutines to handle sync tasks
	openRoutinesHandleTaskAndWaitForFinish()

	for times := 0; times < c.retries; times++ {
		if c.failedTaskGenerateList.Len() != 0 {
			c.urlPairList.PushBackList(c.failedTaskGenerateList)
			c.failedTaskGenerateList.Init()
			// retry to generate task
			fmt.Println("Start to retry to generate sync tasks, please wait ...")
			openRoutinesGenTaskAndWaitForFinish()
		}

		if c.failedTaskList.Len() != 0 {
			c.taskList.PushBackList(c.failedTaskList)
			c.failedTaskList.Init()
		}

		if c.taskList.Len() != 0 {
			// retry to handle task
			fmt.Println("Start to retry sync tasks, please wait ...")
			openRoutinesHandleTaskAndWaitForFinish()
		}
	}

	sync.SynchronizedBlobs.Flush()

	fmt.Printf("Finished, %v sync tasks failed, %v tasks generate failed\n", c.failedTaskList.Len(), c.failedTaskGenerateList.Len())
	c.logger.Infof("Finished, %v sync tasks failed, %v tasks generate failed", c.failedTaskList.Len(), c.failedTaskGenerateList.Len())
}

func (c *Client) GenerateSyncTask(source string, destination string) ([]*UrlPair, error) {
	if source == "" {
		return nil, fmt.Errorf("source url should not be empty")
	}

	sourceUrl, err := tools.NewRepoUrl(source)
	if err != nil {
		return nil, fmt.Errorf("url %s format error: %v", source, err)
	}

	// if dest is not specific, use default registry and namespace
	if destination == "" {
		if c.config.defaultDestRegistry != "" && c.config.defaultDestNamespace != "" {
			destination = c.config.defaultDestRegistry + "/" + c.config.defaultDestNamespace + "/" + sourceUrl.GetRepoWithTag()
		} else {
			return nil, fmt.Errorf("the default registry and namespace should not be nil if you want to use them")
		}
	}

	destUrl, err := tools.NewRepoUrl(destination)
	if err != nil {
		return nil, fmt.Errorf("url %s format error: %v", destination, err)
	}

	// multi-tags config
	tags := sourceUrl.GetTag()
	if moreTag := strings.Split(tags, ","); len(moreTag) > 1 {
		if destUrl.GetTag() != "" && destUrl.GetTag() != sourceUrl.GetTag() {
			return nil, fmt.Errorf("multi-tags source should not correspond to a destination with tag: %s:%s", sourceUrl.GetUrl(), destUrl.GetUrl())
		}

		// contains more than one tag
		var urlPairs = []*UrlPair{}
		for _, t := range moreTag {
			urlPairs = append(urlPairs, &UrlPair{
				source:      sourceUrl.GetUrlWithoutTag() + ":" + t,
				destination: destUrl.GetUrlWithoutTag() + ":" + t,
			})
		}

		return urlPairs, nil
	}

	var imageSource *sync.ImageSource = nil
	var imageDestination *sync.ImageDestination = nil

	if auth, exist := c.config.GetAuth(sourceUrl.GetRegistry()); exist {
		imageSource, err = sync.NewImageSource(sourceUrl.GetRegistry(), sourceUrl.GetRepoWithNamespace(), sourceUrl.GetTag(), auth.Username, auth.Password, auth.Insecure)
		if err != nil {
			return nil, fmt.Errorf("generate %s image source error: %v", sourceUrl.GetUrl(), err)
		}
	} else {
		imageSource, err = sync.NewImageSource(sourceUrl.GetRegistry(), sourceUrl.GetRepoWithNamespace(), sourceUrl.GetTag(), "", "", false)
		if err != nil {
			return nil, fmt.Errorf("generate %s image source error: %v", sourceUrl.GetUrl(), err)
		}
	}

	// if tag is not specific, return tags
	if sourceUrl.GetTag() == "" {
		if destUrl.GetTag() != "" {
			return nil, fmt.Errorf("tag should be included both side of the config: %s:%s", sourceUrl.GetUrl(), destUrl.GetUrl())
		}

		// get all tags of this source repo
		tags, err := imageSource.GetSourceRepoTags()
		if err != nil {
			return nil, fmt.Errorf("get tags failed from %s error: %v", sourceUrl.GetUrl(), err)
		}
		c.logger.Infof("Get tags of %s successfully: %v", sourceUrl.GetUrl(), tags)

		// generate url pairs for tags
		var urlPairs = []*UrlPair{}
		for _, tag := range tags {
			urlPairs = append(urlPairs, &UrlPair{
				source:      sourceUrl.GetUrl() + ":" + tag,
				destination: destUrl.GetUrl() + ":" + tag,
			})
		}
		return urlPairs, nil
	}

	// if source tag is set but without destinate tag, use the same tag as source
	destTag := destUrl.GetTag()
	if destTag == "" {
		destTag = sourceUrl.GetTag()
	}

	if auth, exist := c.config.GetAuth(destUrl.GetRegistry()); exist {
		imageDestination, err = sync.NewImageDestination(destUrl.GetRegistry(), destUrl.GetRepoWithNamespace(), destTag, auth.Username, auth.Password, auth.Insecure)
		if err != nil {
			return nil, fmt.Errorf("generate %s image destination error: %v", sourceUrl.GetUrl(), err)
		}
	} else {
		imageDestination, err = sync.NewImageDestination(destUrl.GetRegistry(), destUrl.GetRepoWithNamespace(), destTag, "", "", false)
		if err != nil {
			return nil, fmt.Errorf("generate %s image destination error: %v", destUrl.GetUrl(), err)
		}
	}

	c.PutATask(sync.NewSyncTask(imageSource, imageDestination, c.logger))
	c.logger.Infof("Generate a task for %s to %s", sourceUrl.GetUrl(), destUrl.GetUrl())
	return nil, nil
}

// return a SyncTask struct and if the task list is empty
func (c *Client) GetATask() (*sync.SyncTask, bool) {
	c.taskListChan <- 1
	task := c.taskList.Front()
	if task == nil {
		<-c.taskListChan
		return nil, true
	}
	c.taskList.Remove(task)
	<-c.taskListChan
	return task.Value.(*sync.SyncTask), false
}

func (c *Client) PutATask(task *sync.SyncTask) {
	c.taskListChan <- 1
	if c.taskList != nil {
		c.taskList.PushBack(task)
	}
	<-c.taskListChan
}

func (c *Client) GetAUrlPair() (*UrlPair, bool) {
	c.urlPairListChan <- 1
	urlPair := c.urlPairList.Front()
	if urlPair == nil {
		<-c.urlPairListChan
		return nil, true
	}
	c.urlPairList.Remove(urlPair)
	<-c.urlPairListChan
	return urlPair.Value.(*UrlPair), false
}

func (c *Client) PutUrlPairs(urlPairs []*UrlPair) {
	c.urlPairListChan <- 1
	if c.urlPairList != nil {
		for _, urlPair := range urlPairs {
			c.urlPairList.PushBack(urlPair)
		}
	}
	<-c.urlPairListChan
}

func (c *Client) GetAFailedTask() (*sync.SyncTask, bool) {
	c.failedTaskListChan <- 1
	failedTask := c.failedTaskList.Front()
	if failedTask == nil {
		<-c.failedTaskListChan
		return nil, true
	}
	c.failedTaskList.Remove(failedTask)
	<-c.failedTaskListChan
	return failedTask.Value.(*sync.SyncTask), false
}

func (c *Client) PutAFailedTask(failedTask *sync.SyncTask) {
	c.failedTaskListChan <- 1
	if c.failedTaskList != nil {
		c.failedTaskList.PushBack(failedTask)
	}
	<-c.failedTaskListChan
}

func (c *Client) GetAFailedUrlPair() (*UrlPair, bool) {
	c.failedTaskGenerateListChan <- 1
	failedUrlPair := c.failedTaskGenerateList.Front()
	if failedUrlPair == nil {
		<-c.failedTaskGenerateListChan
		return nil, true
	}
	c.failedTaskGenerateList.Remove(failedUrlPair)
	<-c.failedTaskGenerateListChan
	return failedUrlPair.Value.(*UrlPair), false
}

func (c *Client) PutAFailedUrlPair(failedUrlPair *UrlPair) {
	c.failedTaskGenerateListChan <- 1
	if c.failedTaskGenerateList != nil {
		c.failedTaskGenerateList.PushBack(failedUrlPair)
	}
	<-c.failedTaskGenerateListChan
}
