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
	TaskList *list.List

	// a URLPair list
	UrlPairList *list.List

	// failed list
	FailedTaskList         *list.List
	FailedTaskGenerateList *list.List

	Config *Config

	RoutineNum int
	Retries    int
	Logger     *logrus.Logger

	// mutex
	TaskListChan               chan int
	UrlPairListChan            chan int
	FailedTaskListChan         chan int
	FailedTaskGenerateListChan chan int
}

// URLPair is a pair of source and destination url
type URLPair struct {
	source      string
	destination string
}

// NewSyncClient creates a synchronization client
func NewSyncClient(configFile, authFile, imageFile, logFile string,
	routineNum, retries int, defaultDestRegistry, defaultDestNamespace string,
	osFilterList, archFilterList []string) (*Client, error) {

	logger := NewFileLogger(logFile)

	config, err := NewSyncConfig(configFile, authFile, imageFile,
		defaultDestRegistry, defaultDestNamespace, osFilterList, archFilterList)
	if err != nil {
		return nil, fmt.Errorf("generate config error: %v", err)
	}

	return &Client{
		TaskList:                   list.New(),
		UrlPairList:                list.New(),
		FailedTaskList:             list.New(),
		FailedTaskGenerateList:     list.New(),
		Config:                     config,
		RoutineNum:                 routineNum,
		Retries:                    retries,
		Logger:                     logger,
		TaskListChan:               make(chan int, 1),
		UrlPairListChan:            make(chan int, 1),
		FailedTaskListChan:         make(chan int, 1),
		FailedTaskGenerateListChan: make(chan int, 1),
	}, nil
}

// Run is main function of a synchronization client
func (c *Client) Run() {
	fmt.Println("Start to generate sync tasks, please wait ...")

	//var finishChan = make(chan struct{}, c.routineNum)

	// open num of goroutines and wait c for close
	openRoutinesGenTaskAndWaitForFinish := func() {
		wg := sync2.WaitGroup{}
		for i := 0; i < c.RoutineNum; i++ {
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
						c.Logger.Errorf("Generate sync task %s to %s error: %v", urlPair.source, urlPair.destination, err)
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
		for i := 0; i < c.RoutineNum; i++ {
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

	for source, dest := range c.Config.GetImageList() {
		c.UrlPairList.PushBack(&URLPair{
			source:      source,
			destination: dest,
		})
	}

	// generate sync tasks
	openRoutinesGenTaskAndWaitForFinish()

	fmt.Println("Start to handle sync tasks, please wait ...")

	// generate goroutines to handle sync tasks
	openRoutinesHandleTaskAndWaitForFinish()

	for times := 0; times < c.Retries; times++ {
		if c.FailedTaskGenerateList.Len() != 0 {
			c.UrlPairList.PushBackList(c.FailedTaskGenerateList)
			c.FailedTaskGenerateList.Init()
			// retry to generate task
			fmt.Println("Start to retry to generate sync tasks, please wait ...")
			openRoutinesGenTaskAndWaitForFinish()
		}

		if c.FailedTaskList.Len() != 0 {
			c.TaskList.PushBackList(c.FailedTaskList)
			c.FailedTaskList.Init()
		}

		if c.TaskList.Len() != 0 {
			// retry to handle task
			fmt.Println("Start to retry sync tasks, please wait ...")
			openRoutinesHandleTaskAndWaitForFinish()
		}
	}

	fmt.Printf("Finished, %v sync tasks failed, %v tasks generate failed\n", c.FailedTaskList.Len(), c.FailedTaskGenerateList.Len())
	c.Logger.Infof("Finished, %v sync tasks failed, %v tasks generate failed", c.FailedTaskList.Len(), c.FailedTaskGenerateList.Len())
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
		if c.Config.defaultDestRegistry != "" && c.Config.defaultDestNamespace != "" {
			destination = c.Config.defaultDestRegistry + "/" + c.Config.defaultDestNamespace + "/" +
				sourceURL.GetRepoWithTag()
		} else {
			return nil, fmt.Errorf("the default registry and namespace should not be nil if you want to use them")
		}
	}

	destURL, err := tools.NewRepoURL(destination)
	if err != nil {
		return nil, fmt.Errorf("url %s format error: %v", destination, err)
	}

	tags := sourceURL.GetTag()

	// multi-tags config
	if moreTag := strings.Split(tags, ","); len(moreTag) > 1 {
		if destURL.GetTag() != "" && destURL.GetTag() != sourceURL.GetTag() {
			return nil, fmt.Errorf("multi-tags source should not correspond to a destination with tag: %s:%s",
				sourceURL.GetURL(), destURL.GetURL())
		}

		// contains more than one tag
		var urlPairs []*URLPair
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

	if auth, exist := c.Config.GetAuth(sourceURL.GetRegistry(), sourceURL.GetNamespace()); exist {
		c.Logger.Infof("Find auth information for %v, username: %v", sourceURL.GetURL(), auth.Username)
		imageSource, err = sync.NewImageSource(sourceURL.GetRegistry(), sourceURL.GetRepoWithNamespace(), sourceURL.GetTag(),
			auth.Username, auth.Password, auth.Insecure)
		if err != nil {
			return nil, fmt.Errorf("generate %s image source error: %v", sourceURL.GetURL(), err)
		}
	} else {
		c.Logger.Infof("Cannot find auth information for %v, pull actions will be anonymous", sourceURL.GetURL())
		imageSource, err = sync.NewImageSource(sourceURL.GetRegistry(), sourceURL.GetRepoWithNamespace(), sourceURL.GetTag(),
			"", "", false)
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
		c.Logger.Infof("Get tags of %s successfully: %v", sourceURL.GetURL(), tags)

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

	// if source tag is set but without destination tag, use the same tag as source
	destTag := destURL.GetTag()
	if destTag == "" {
		destTag = sourceURL.GetTag()
	}

	if auth, exist := c.Config.GetAuth(destURL.GetRegistry(), destURL.GetNamespace()); exist {
		c.Logger.Infof("Find auth information for %v, username: %v", destURL.GetURL(), auth.Username)
		imageDestination, err = sync.NewImageDestination(destURL.GetRegistry(), destURL.GetRepoWithNamespace(),
			destTag, auth.Username, auth.Password, auth.Insecure)
		if err != nil {
			return nil, fmt.Errorf("generate %s image destination error: %v", sourceURL.GetURL(), err)
		}
	} else {
		c.Logger.Infof("Cannot find auth information for %v, push actions will be anonymous", destURL.GetURL())
		imageDestination, err = sync.NewImageDestination(destURL.GetRegistry(), destURL.GetRepoWithNamespace(),
			destTag, "", "", false)
		if err != nil {
			return nil, fmt.Errorf("generate %s image destination error: %v", destURL.GetURL(), err)
		}
	}

	c.PutATask(sync.NewTask(imageSource, imageDestination, c.Config.osFilterList, c.Config.archFilterList, c.Logger))
	c.Logger.Infof("Generate a task for %s to %s", sourceURL.GetURL(), destURL.GetURL())
	return nil, nil
}

// GetATask return a sync.Task struct if the task list is not empty
func (c *Client) GetATask() (*sync.Task, bool) {
	c.TaskListChan <- 1
	defer func() {
		<-c.TaskListChan
	}()

	task := c.TaskList.Front()
	if task == nil {
		return nil, true
	}
	c.TaskList.Remove(task)

	return task.Value.(*sync.Task), false
}

// PutATask puts a sync.Task struct to task list
func (c *Client) PutATask(task *sync.Task) {
	c.TaskListChan <- 1
	defer func() {
		<-c.TaskListChan
	}()

	if c.TaskList != nil {
		c.TaskList.PushBack(task)
	}
}

// GetAURLPair gets a URLPair from urlPairList
func (c *Client) GetAURLPair() (*URLPair, bool) {
	c.UrlPairListChan <- 1
	defer func() {
		<-c.UrlPairListChan
	}()

	urlPair := c.UrlPairList.Front()
	if urlPair == nil {
		return nil, true
	}
	c.UrlPairList.Remove(urlPair)

	return urlPair.Value.(*URLPair), false
}

// PutURLPairs puts a URLPair array to urlPairList
func (c *Client) PutURLPairs(urlPairs []*URLPair) {
	c.UrlPairListChan <- 1
	defer func() {
		<-c.UrlPairListChan
	}()

	if c.UrlPairList != nil {
		for _, urlPair := range urlPairs {
			c.UrlPairList.PushBack(urlPair)
		}
	}
}

// GetAFailedTask gets a failed task from failedTaskList
func (c *Client) GetAFailedTask() (*sync.Task, bool) {
	c.FailedTaskListChan <- 1
	defer func() {
		<-c.FailedTaskListChan
	}()

	failedTask := c.FailedTaskList.Front()
	if failedTask == nil {
		return nil, true
	}
	c.FailedTaskList.Remove(failedTask)

	return failedTask.Value.(*sync.Task), false
}

// PutAFailedTask puts a failed task to failedTaskList
func (c *Client) PutAFailedTask(failedTask *sync.Task) {
	c.FailedTaskListChan <- 1
	defer func() {
		<-c.FailedTaskListChan
	}()

	if c.FailedTaskList != nil {
		c.FailedTaskList.PushBack(failedTask)
	}
}

// GetAFailedURLPair get a URLPair from failedTaskGenerateList
func (c *Client) GetAFailedURLPair() (*URLPair, bool) {
	c.FailedTaskGenerateListChan <- 1
	defer func() {
		<-c.FailedTaskGenerateListChan
	}()

	failedURLPair := c.FailedTaskGenerateList.Front()
	if failedURLPair == nil {
		return nil, true
	}
	c.FailedTaskGenerateList.Remove(failedURLPair)

	return failedURLPair.Value.(*URLPair), false
}

// PutAFailedURLPair puts a URLPair to failedTaskGenerateList
func (c *Client) PutAFailedURLPair(failedURLPair *URLPair) {
	c.FailedTaskGenerateListChan <- 1
	defer func() {
		<-c.FailedTaskGenerateListChan
	}()

	if c.FailedTaskGenerateList != nil {
		c.FailedTaskGenerateList.PushBack(failedURLPair)
	}
}
