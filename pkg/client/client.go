package client

import (
	"container/list"
	"fmt"
	"strings"
	sync2 "sync"

	"github.com/AliyunContainerService/image-syncer/pkg/sync"
	"github.com/AliyunContainerService/image-syncer/pkg/tools"
	"github.com/sirupsen/logrus"
)

// Client describes a synchronization client
type Client struct {
	// a sync.Task list
	taskList *list.List

	// a URLPair list
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

// URLPair is a pair of source and destination url
type URLPair struct {
	source      string
	destination string
}

// NewSyncClient creates a synchronization client
func NewSyncClient(configFile, logFile, recordsFile string, routineNum, retries int, defaultDestRegistry string, defaultDestNamespace string) (*Client, error) {
	logger := NewFileLogger(logFile)

	config, err := NewSyncConfig(configFile, defaultDestRegistry, defaultDestNamespace)
	if err != nil {
		return nil, fmt.Errorf("Generate confg error: %v", err)
	}

	// init blob recorder
	if err := sync.NewSynchronizedBlobRecorder(recordsFile); err != nil {
		return nil, err
	}

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

// Run is main function of a synchronization client
func (c *Client) Run() {
	fmt.Println("Start to generate sync tasks, please wait ...")

	//var finishChan = make(chan struct{}, c.routineNum)

	// opem num of goroutines and wait c for close
	openRoutinesGenTaskAndWaitForFinish := func() {
		wg := sync2.WaitGroup{}
		for i := 0; i < c.routineNum; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					urlPair, empty := c.GetAURLPair()
					// no more task to generate
					if empty {
						break
					}
					moreURLPairs, err := c.GenerateSyncTask(urlPair.source, urlPair.destination)
					if err != nil {
						c.logger.Errorf("Generate sync task %s to %s error: %v", urlPair.source, urlPair.destination, err)
						// put to failedTaskGenerateList
						c.PutAFailedURLPair(urlPair)
					}
					if moreURLPairs != nil {
						c.PutURLPairs(moreURLPairs)
					}
				}
			}()
		}
		wg.Wait()
	}

	openRoutinesHandleTaskAndWaitForFinish := func() {
		wg := sync2.WaitGroup{}
		for i := 0; i < c.routineNum; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
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
			}()
		}

		wg.Wait()
	}

	for source, dest := range c.config.GetImageList() {
		c.urlPairList.PushBack(&URLPair{
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

// GenerateSyncTask creates synchronization tasks from source and destination url, return URLPair array if there are more than one tags
func (c *Client) GenerateSyncTask(source string, destination string) ([]*URLPair, error) {
	if source == "" {
		return nil, fmt.Errorf("source url should not be empty")
	}

	sourceURL, err := tools.NewRepoURL(source)
	if err != nil {
		return nil, fmt.Errorf("url %s format error: %v", source, err)
	}

	// if dest is not specific, use default registry and namespace
	if destination == "" {
		if c.config.defaultDestRegistry != "" && c.config.defaultDestNamespace != "" {
			destination = c.config.defaultDestRegistry + "/" + c.config.defaultDestNamespace + "/" + sourceURL.GetRepoWithTag()
		} else {
			return nil, fmt.Errorf("the default registry and namespace should not be nil if you want to use them")
		}
	}

	destURL, err := tools.NewRepoURL(destination)
	if err != nil {
		return nil, fmt.Errorf("url %s format error: %v", destination, err)
	}

	// multi-tags config
	tags := sourceURL.GetTag()
	if moreTag := strings.Split(tags, ","); len(moreTag) > 1 {
		if destURL.GetTag() != "" && destURL.GetTag() != sourceURL.GetTag() {
			return nil, fmt.Errorf("multi-tags source should not correspond to a destination with tag: %s:%s", sourceURL.GetURL(), destURL.GetURL())
		}

		// contains more than one tag
		var urlPairs = []*URLPair{}
		for _, t := range moreTag {
			urlPairs = append(urlPairs, &URLPair{
				source:      sourceURL.GetURLWithoutTag() + ":" + t,
				destination: destURL.GetURLWithoutTag() + ":" + t,
			})
		}

		return urlPairs, nil
	}

	var imageSource *sync.ImageSource
	var imageDestination *sync.ImageDestination

	if auth, exist := c.config.GetAuth(sourceURL.GetRegistry(), sourceURL.GetNamespace()); exist {
		c.logger.Infof("Find auth information for %v, username: %v", sourceURL.GetURL(), auth.Username)
		imageSource, err = sync.NewImageSource(sourceURL.GetRegistry(), sourceURL.GetRepoWithNamespace(), sourceURL.GetTag(), auth.Username, auth.Password, auth.Insecure)
		if err != nil {
			return nil, fmt.Errorf("generate %s image source error: %v", sourceURL.GetURL(), err)
		}
	} else {
		c.logger.Infof("Cannot find auth information for %v, pull actions will be anonymous", sourceURL.GetURL())
		imageSource, err = sync.NewImageSource(sourceURL.GetRegistry(), sourceURL.GetRepoWithNamespace(), sourceURL.GetTag(), "", "", false)
		if err != nil {
			return nil, fmt.Errorf("generate %s image source error: %v", sourceURL.GetURL(), err)
		}
	}

	// if tag is not specific, return tags
	if sourceURL.GetTag() == "" {
		if destURL.GetTag() != "" {
			return nil, fmt.Errorf("tag should be included both side of the config: %s:%s", sourceURL.GetURL(), destURL.GetURL())
		}

		// get all tags of this source repo
		tags, err := imageSource.GetSourceRepoTags()
		if err != nil {
			return nil, fmt.Errorf("get tags failed from %s error: %v", sourceURL.GetURL(), err)
		}
		c.logger.Infof("Get tags of %s successfully: %v", sourceURL.GetURL(), tags)

		// generate url pairs for tags
		var urlPairs = []*URLPair{}
		for _, tag := range tags {
			urlPairs = append(urlPairs, &URLPair{
				source:      sourceURL.GetURL() + ":" + tag,
				destination: destURL.GetURL() + ":" + tag,
			})
		}
		return urlPairs, nil
	}

	// if source tag is set but without destinate tag, use the same tag as source
	destTag := destURL.GetTag()
	if destTag == "" {
		destTag = sourceURL.GetTag()
	}

	if auth, exist := c.config.GetAuth(destURL.GetRegistry(), destURL.GetNamespace()); exist {
		c.logger.Infof("Find auth information for %v, username: %v", destURL.GetURL(), auth.Username)
		imageDestination, err = sync.NewImageDestination(destURL.GetRegistry(), destURL.GetRepoWithNamespace(), destTag, auth.Username, auth.Password, auth.Insecure)
		if err != nil {
			return nil, fmt.Errorf("generate %s image destination error: %v", sourceURL.GetURL(), err)
		}
	} else {
		c.logger.Infof("Cannot find auth information for %v, push actions will be anonymous", destURL.GetURL())
		imageDestination, err = sync.NewImageDestination(destURL.GetRegistry(), destURL.GetRepoWithNamespace(), destTag, "", "", false)
		if err != nil {
			return nil, fmt.Errorf("generate %s image destination error: %v", destURL.GetURL(), err)
		}
	}

	c.PutATask(sync.NewTask(imageSource, imageDestination, c.logger))
	c.logger.Infof("Generate a task for %s to %s", sourceURL.GetURL(), destURL.GetURL())
	return nil, nil
}

// GetATask return a sync.Task struct if the task list is not empty
func (c *Client) GetATask() (*sync.Task, bool) {
	c.taskListChan <- 1
	task := c.taskList.Front()
	if task == nil {
		<-c.taskListChan
		return nil, true
	}
	c.taskList.Remove(task)
	<-c.taskListChan
	return task.Value.(*sync.Task), false
}

// PutATask puts a sync.Task struct to task list
func (c *Client) PutATask(task *sync.Task) {
	c.taskListChan <- 1
	if c.taskList != nil {
		c.taskList.PushBack(task)
	}
	<-c.taskListChan
}

// GetAURLPair gets a URLPair from urlPairList
func (c *Client) GetAURLPair() (*URLPair, bool) {
	c.urlPairListChan <- 1
	urlPair := c.urlPairList.Front()
	if urlPair == nil {
		<-c.urlPairListChan
		return nil, true
	}
	c.urlPairList.Remove(urlPair)
	<-c.urlPairListChan
	return urlPair.Value.(*URLPair), false
}

// PutURLPairs puts a URLPair array to urlPairList
func (c *Client) PutURLPairs(urlPairs []*URLPair) {
	c.urlPairListChan <- 1
	if c.urlPairList != nil {
		for _, urlPair := range urlPairs {
			c.urlPairList.PushBack(urlPair)
		}
	}
	<-c.urlPairListChan
}

// GetAFailedTask gets a failed task from failedTaskList
func (c *Client) GetAFailedTask() (*sync.Task, bool) {
	c.failedTaskListChan <- 1
	failedTask := c.failedTaskList.Front()
	if failedTask == nil {
		<-c.failedTaskListChan
		return nil, true
	}
	c.failedTaskList.Remove(failedTask)
	<-c.failedTaskListChan
	return failedTask.Value.(*sync.Task), false
}

// PutAFailedTask puts a failed task to failedTaskList
func (c *Client) PutAFailedTask(failedTask *sync.Task) {
	c.failedTaskListChan <- 1
	if c.failedTaskList != nil {
		c.failedTaskList.PushBack(failedTask)
	}
	<-c.failedTaskListChan
}

// GetAFailedURLPair get a URLPair from failedTaskGenerateList
func (c *Client) GetAFailedURLPair() (*URLPair, bool) {
	c.failedTaskGenerateListChan <- 1
	failedURLPair := c.failedTaskGenerateList.Front()
	if failedURLPair == nil {
		<-c.failedTaskGenerateListChan
		return nil, true
	}
	c.failedTaskGenerateList.Remove(failedURLPair)
	<-c.failedTaskGenerateListChan
	return failedURLPair.Value.(*URLPair), false
}

// PutAFailedURLPair puts a URLPair to failedTaskGenerateList
func (c *Client) PutAFailedURLPair(failedURLPair *URLPair) {
	c.failedTaskGenerateListChan <- 1
	if c.failedTaskGenerateList != nil {
		c.failedTaskGenerateList.PushBack(failedURLPair)
	}
	<-c.failedTaskGenerateListChan
}
