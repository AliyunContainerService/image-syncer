package client

import (
	"fmt"

	"github.com/AliyunContainerService/image-syncer/pkg/task"

	"github.com/AliyunContainerService/image-syncer/pkg/concurrent"

	"github.com/AliyunContainerService/image-syncer/pkg/sync"
	"github.com/AliyunContainerService/image-syncer/pkg/utils"
	"github.com/sirupsen/logrus"
)

// Client describes a synchronization client
type Client struct {
	taskList       *concurrent.List
	failedTaskList *concurrent.List

	urlPairList                   *concurrent.List
	failedGenerateTaskUrlPairList *concurrent.List

	taskCounter         *concurrent.Counter
	generateTaskCounter *concurrent.Counter

	failedTaskCounter         *concurrent.Counter
	failedGenerateTaskCounter *concurrent.Counter

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

func (u *URLPair) String() string {
	return u.source + "->" + u.destination
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
		taskList:                      concurrent.NewList(),
		urlPairList:                   concurrent.NewList(),
		failedTaskList:                concurrent.NewList(),
		failedGenerateTaskUrlPairList: concurrent.NewList(),
		generateTaskCounter:           concurrent.NewCounter(0, 0),
		failedGenerateTaskCounter:     concurrent.NewCounter(0, 0),
		taskCounter:                   concurrent.NewCounter(0, 0),
		failedTaskCounter:             concurrent.NewCounter(0, 0),
		config:                        config,
		routineNum:                    routineNum,
		retries:                       retries,
		logger:                        logger,
	}, nil
}

// Run is main function of a synchronization client
func (c *Client) Run() error {
	c.logger.Infof("Start to generate sync tasks, please wait ...")

	for source, dest := range c.config.GetImageList() {
		c.urlPairList.PushBack(&URLPair{
			source:      source,
			destination: dest,
		})
		c.generateTaskCounter.IncreaseTotal()
	}

	// generate sync tasks
	c.openRoutinesGenTaskAndWaitForFinish()

	c.logger.Infof("Start to handle sync tasks, please wait ...")

	// generate goroutines to handle sync tasks
	c.openRoutinesHandleTaskAndWaitForFinish()

	for times := 0; times < c.retries; times++ {
		c.taskCounter, c.failedTaskCounter = c.failedTaskCounter, concurrent.NewCounter(0, 0)
		c.generateTaskCounter, c.failedGenerateTaskCounter = c.failedGenerateTaskCounter,
			concurrent.NewCounter(0, 0)

		if c.failedGenerateTaskUrlPairList.Len() != 0 {
			c.urlPairList.PushBackList(c.failedGenerateTaskUrlPairList)
			c.failedGenerateTaskUrlPairList.Reset()

			// retry to generate task
			c.logger.Infof("Start to retry to generate sync tasks, please wait ...")
			c.openRoutinesGenTaskAndWaitForFinish()
		}

		if c.failedTaskList.Len() != 0 {
			c.taskList.PushBackList(c.failedTaskList)
			c.failedTaskList.Reset()
		}

		if c.taskList.Len() != 0 {
			// retry to handle task
			c.logger.Infof("Start to retry sync tasks, please wait ...")
			c.openRoutinesHandleTaskAndWaitForFinish()
		}
	}

	endMsg := fmt.Sprintf("Finished, %v tasks failed, %v rules failed to be generated to tasks",
		c.failedTaskList.Len(), c.failedGenerateTaskUrlPairList.Len())

	c.logger.Infof(utils.Green(endMsg))

	_, failedTaskCountTotal := c.failedTaskCounter.Value()
	_, failedGenerateTaskCountTotal := c.failedGenerateTaskCounter.Value()

	if failedTaskCountTotal != 0 || failedGenerateTaskCountTotal != 0 {
		return fmt.Errorf(endMsg)
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

			c.logger.Infof("Generating tasks for %v...", urlPair.(*URLPair).String())

			if err := c.GenerateSyncTasks(urlPair.(*URLPair).source, urlPair.(*URLPair).destination); err != nil {
				c.logger.Errorf("Generate sync task %s error: %v",
					urlPair.(*URLPair).String(), err)

				// put to failedTaskGenerateList
				c.failedGenerateTaskUrlPairList.PushBack(urlPair)
				c.failedGenerateTaskCounter.IncreaseTotal()
			}

			count, total := c.generateTaskCounter.Increase()
			finishedNumString := utils.Green(fmt.Sprintf("%d", count))
			totalNumString := utils.Green(fmt.Sprintf("%d", total))

			c.logger.Infof("Finish generating tasks for %v, %v/%v generate tasks executed", urlPair.(*URLPair).String(),
				finishedNumString, totalNumString)
		}
	})
}

func (c *Client) openRoutinesHandleTaskAndWaitForFinish() {
	concurrent.CreateRoutinesAndWaitForFinish(c.routineNum, func() {
		for {
			item := c.taskList.PopFront()
			// no more tasks need to handle
			if item == nil {
				break
			}
			syncTask := item.(task.Task)

			c.logger.Infof("Executing %v...", syncTask.String())

			primary, message, err := syncTask.Run()
			if err != nil {
				c.failedTaskList.PushBack(syncTask)
				c.failedTaskCounter.IncreaseTotal()
			}

			count, total := c.taskCounter.Increase()
			finishedNumString := utils.Green(fmt.Sprintf("%d", count))
			totalNumString := utils.Green(fmt.Sprintf("%d", total))

			if len(message) != 0 {
				c.logger.Infof("Finish %v: %v, %v/%v tasks executed", syncTask.String(), message,
					finishedNumString, totalNumString)
			} else {
				c.logger.Infof("Finish %v, %v/%v tasks executed", syncTask.String(),
					finishedNumString, totalNumString)
			}

			if primary != nil {
				// handler manifest
				c.taskList.PushFront(primary)
				c.taskCounter.IncreaseTotal()
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

	for _, t := range tasks {
		c.taskList.PushBack(t)
		c.taskCounter.IncreaseTotal()
	}

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

func (c *Client) generateTasks(sourceURLs, destinationURLs []*utils.RepoURL) ([]task.Task, error) {
	var result []task.Task
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

		tasks, msg, err := task.GenerateTasks(imageSource, imageDestination, c.config.osFilterList, c.config.archFilterList)
		if err != nil {
			return nil, fmt.Errorf("failed to generate tasks: %v", err)
		}

		if len(tasks) == 0 {
			c.logger.Infof("No task is generated for %s->%s: %v", imageSource, imageDestination, msg)
		}

		result = append(result, tasks...)
	}

	return result, nil
}
