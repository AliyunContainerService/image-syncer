package client

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/AliyunContainerService/image-syncer/pkg/concurrent"
	"github.com/AliyunContainerService/image-syncer/pkg/sync"
	"github.com/AliyunContainerService/image-syncer/pkg/utils"
)

// Client describes a synchronization client
type Client struct {
	// a sync.Task list
	taskList *concurrent.List

	// a URLPair list
	urlPairList *concurrent.List

	// failed list
	failedTaskList         *concurrent.List
	failedTaskGenerateList *concurrent.List

	config *Config

	routineNum int
	retries    int
	logger     *logrus.Logger
}

// URLPair is a pair of source and destination url
type URLPair struct {
	source      string
	destination string
}

// NewSyncClient creates a synchronization client
func NewSyncClient(configFile, authFile, imageFile, logFile string,
	routineNum, retries int, defaultDestRegistry string,
	osFilterList, archFilterList []string) (*Client, error) {

	logger := NewFileLogger(logFile)

	config, err := NewSyncConfig(configFile, authFile, imageFile,
		defaultDestRegistry, osFilterList, archFilterList)
	if err != nil {
		return nil, fmt.Errorf("generate config error: %v", err)
	}

	return &Client{
		taskList:               concurrent.NewList(),
		urlPairList:            concurrent.NewList(),
		failedTaskList:         concurrent.NewList(),
		failedTaskGenerateList: concurrent.NewList(),
		config:                 config,
		routineNum:             routineNum,
		retries:                retries,
		logger:                 logger,
	}, nil
}

// Run is main function of a synchronization client
func (c *Client) Run() error {
	fmt.Println("Start to generate sync tasks, please wait ...")

	for source, dest := range c.config.GetImageList() {
		c.urlPairList.PushBack(&URLPair{
			source:      source,
			destination: dest,
		})
	}

	// generate sync tasks
	c.openRoutinesGenTaskAndWaitForFinish()

	fmt.Println("Start to handle sync tasks, please wait ...")

	// generate goroutines to handle sync tasks
	c.openRoutinesHandleTaskAndWaitForFinish()

	for times := 0; times < c.retries; times++ {
		if c.failedTaskGenerateList.Len() != 0 {
			c.urlPairList.PushBackList(c.failedTaskGenerateList)
			c.failedTaskGenerateList.Reset()
			// retry to generate task
			fmt.Println("Start to retry to generate sync tasks, please wait ...")
			c.openRoutinesGenTaskAndWaitForFinish()
		}

		if c.failedTaskList.Len() != 0 {
			c.taskList.PushBackList(c.failedTaskList)
			c.failedTaskList.Reset()
		}

		if c.taskList.Len() != 0 {
			// retry to handle task
			fmt.Println("Start to retry sync tasks, please wait ...")
			c.openRoutinesHandleTaskAndWaitForFinish()
		}
	}

	fmt.Printf("Finished, %v sync tasks failed, %v tasks generate failed\n", c.failedTaskList.Len(), c.failedTaskGenerateList.Len())
	c.logger.Infof("Finished, %v sync tasks failed, %v tasks generate failed", c.failedTaskList.Len(), c.failedTaskGenerateList.Len())

	if c.failedTaskList.Len() > 0 || c.failedTaskGenerateList.Len() > 0 {
		return fmt.Errorf("%v sync tasks failed, %v tasks generate failed", c.failedTaskList.Len(), c.failedTaskGenerateList.Len())
	}

	return nil
}

func (c *Client) openRoutinesGenTaskAndWaitForFinish() {
	concurrent.CreateRoutinesAndWaitForFinish(c.routineNum, func() {
		for {
			urlPair := c.urlPairList.PopFront()
			// no more task to generate
			if urlPair == nil {
				break
			}
			if err := c.GenerateSyncTasks(urlPair.(*URLPair).source, urlPair.(*URLPair).destination); err != nil {
				c.logger.Errorf("Generate sync task %s to %s error: %v",
					urlPair.(*URLPair).source, urlPair.(*URLPair).destination, err)
				// put to failedTaskGenerateList
				c.failedTaskList.PushBack(urlPair)
			}
		}
	})
}

func (c *Client) openRoutinesHandleTaskAndWaitForFinish() {
	concurrent.CreateRoutinesAndWaitForFinish(c.routineNum, func() {
		for {
			task := c.taskList.PopFront()
			// no more tasks need to handle
			if task == nil {
				break
			}
			if err := task.(*sync.Task).Run(); err != nil {
				// put to failedTaskList
				c.taskList.PushBack(task)
			}
		}
	})
}

// GenerateSyncTasks creates synchronization tasks from source and destination url
func (c *Client) GenerateSyncTasks(source string, destination string) error {
	if source == "" {
		return fmt.Errorf("source url should not be empty")
	}

	// if source tag is not specific, get all tags of this source repo
	sourceURLs, err := utils.GenerateRepoURLs(source, c.listAllTags)
	if err != nil {
		return fmt.Errorf("source url %s format error: %v", source, err)
	}

	// if dest is not specific, use default registry and repo
	if destination == "" {
		if c.config.defaultDestRegistry != "" {
			destination = c.config.defaultDestRegistry + "/" +
				sourceURLs[0].GetRepo()
		} else {
			return fmt.Errorf("the default registry and namespace should not be nil if you want to use them")
		}
	}

	// if destination tags or digest is not specific, reuse tags or digest of sourceURLs
	destinationURLs, err := utils.GenerateRepoURLs(destination, func(registry, repository string) ([]string, error) {
		var result []string
		for _, item := range sourceURLs {
			result = append(result, item.GetTagOrDigest())
		}
		return result, nil
	})
	if err != nil {
		return fmt.Errorf("source url %s format error: %v", source, err)
	}

	if err = c.checkSourceAndDestinationURLs(sourceURLs, destinationURLs); err != nil {
		return fmt.Errorf("failed to check source and destination urls for %s:%s: %v", source, destination, err)
	}

	tasks, err := c.generateTasks(sourceURLs, destinationURLs)
	if err != nil {
		return fmt.Errorf("failed to generate tasks: %v", err)
	}

	for _, task := range tasks {
		c.taskList.PushBack(task)
	}

	c.logger.Infof("Generate a task for %s to %s", source, destination)
	return nil
}

func (c *Client) checkSourceAndDestinationURLs(sourceURLs, destinationURLs []*utils.RepoURL) error {
	if len(sourceURLs) != len(destinationURLs) {
		return fmt.Errorf("the number of tags of source and destination is not matched")
	}

	// digest must be the same
	if len(sourceURLs) == 1 && sourceURLs[0].HasDigest() && destinationURLs[0].HasDigest() {
		if sourceURLs[0].GetTagOrDigest() != destinationURLs[0].GetTagOrDigest() {
			return fmt.Errorf("the digest of source and destination must match")
		}
	}

	return nil
}

func (c *Client) listAllTags(registry, repository string) ([]string, error) {
	auth, exist := c.config.GetAuth(registry + "/" + repository)
	if exist {
		c.logger.Infof("Find auth information for %v, username: %v", registry+"/"+repository, auth.Username)
	}
	imageSource, err := sync.NewImageSource(registry, repository, "",
		auth.Username, auth.Password, auth.Insecure)
	if err != nil {
		return nil, fmt.Errorf("generate %s image source error: %v", registry+"/"+repository, err)
	}

	return imageSource.GetSourceRepoTags()
}

func (c *Client) generateTasks(sourceURLs, destinationURLs []*utils.RepoURL) ([]*sync.Task, error) {
	var result []*sync.Task
	for index, s := range sourceURLs {

		auth, exist := c.config.GetAuth(s.GetURLWithoutTagOrDigest())
		if exist {
			c.logger.Infof("Find auth information for %v, username: %v", s.String(), auth.Username)
		}

		imageSource, err := sync.NewImageSource(s.GetRegistry(), s.GetRepo(), s.GetTagOrDigest(),
			auth.Username, auth.Password, auth.Insecure)
		if err != nil {
			return nil, fmt.Errorf("generate %s image source error: %v", s.String(), err)
		}

		auth, exist = c.config.GetAuth(destinationURLs[index].GetURLWithoutTagOrDigest())
		if exist {
			c.logger.Infof("Find auth information for %v, username: %v", destinationURLs[index].String(), auth.Username)
		}

		imageDestination, err := sync.NewImageDestination(destinationURLs[index].GetRegistry(),
			destinationURLs[index].GetRepo(),
			destinationURLs[index].GetTagOrDigest(), auth.Username, auth.Password, auth.Insecure)
		if err != nil {
			return nil, fmt.Errorf("generate %s image destination error: %v", destinationURLs[index].String(), err)
		}

		result = append(result, sync.NewTask(imageSource, imageDestination, c.config.osFilterList, c.config.archFilterList, c.logger))
	}

	return result, nil
}
