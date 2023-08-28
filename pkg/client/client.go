package client

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/AliyunContainerService/image-syncer/pkg/utils/types"
	"gopkg.in/yaml.v2"

	"github.com/AliyunContainerService/image-syncer/pkg/concurrent"
	"github.com/AliyunContainerService/image-syncer/pkg/task"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

// Client describes a synchronization client
type Client struct {
	taskList       *concurrent.List
	failedTaskList *concurrent.List

	taskCounter       *concurrent.Counter
	failedTaskCounter *concurrent.Counter

	// TODO: print failed images? but this might be difficult because image sync rule might be illegal
	successImagesList                     *concurrent.ImageList
	successImagesFile, outputImagesFormat string

	config *Config

	routineNum int
	retries    int
	logger     *logrus.Logger

	forceUpdate bool
}

// NewSyncClient creates a synchronization client
func NewSyncClient(configFile, authFile, imagesFile, logFile, successImagesFile, outputImagesFormat string,
	routineNum, retries int, osFilterList, archFilterList []string, forceUpdate bool) (*Client, error) {

	logger := NewFileLogger(logFile)

	config, err := NewSyncConfig(configFile, authFile, imagesFile, osFilterList, archFilterList, logger)
	if err != nil {
		return nil, fmt.Errorf("generate config error: %v", err)
	}

	return &Client{
		taskList:       concurrent.NewList(),
		failedTaskList: concurrent.NewList(),

		taskCounter:       concurrent.NewCounter(0, 0),
		failedTaskCounter: concurrent.NewCounter(0, 0),

		successImagesList:  concurrent.NewImageList(),
		successImagesFile:  successImagesFile,
		outputImagesFormat: outputImagesFormat,

		config:     config,
		routineNum: routineNum,
		retries:    retries,
		logger:     logger,

		forceUpdate: forceUpdate,
	}, nil
}

// Run is main function of a synchronization client
func (c *Client) Run() error {
	start := time.Now()

	imageList, err := types.NewImageList(c.config.ImageList)
	if err != nil {
		return fmt.Errorf("failed to get image list: %v", err)
	}

	for source, destList := range imageList {
		for _, dest := range destList {
			// TODO: support multiple destinations for one task
			ruleTask, err := task.NewRuleTask(source, dest,
				func(repository string) types.Auth {
					auth, exist := c.config.GetAuth(repository)
					if !exist {
						c.logger.Infof("Auth information not found for %v, access will be anonymous.", repository)
					}
					return auth
				}, c.forceUpdate)
			if err != nil {
				return fmt.Errorf("failed to generate rule task for %s -> %s: %v", source, dest, err)
			}

			c.taskList.PushBack(ruleTask)
			c.taskCounter.IncreaseTotal()
		}
	}

	c.openRoutinesHandleTaskAndWaitForFinish()

	for times := 0; times < c.retries; times++ {
		c.taskCounter, c.failedTaskCounter = c.failedTaskCounter, concurrent.NewCounter(0, 0)

		if c.failedTaskList.Len() != 0 {
			c.taskList.PushBackList(c.failedTaskList)
			c.failedTaskList.Reset()
		}

		if c.taskList.Len() != 0 {
			// retry to handle task
			c.logger.Infof("Start to retry tasks, please wait ...")
			c.openRoutinesHandleTaskAndWaitForFinish()
		}
	}

	endMsg := fmt.Sprintf("Synchronization finished, %v tasks failed, cost %v.",
		c.failedTaskList.Len(), time.Since(start).String())

	c.logger.Infof(color.New(color.FgGreen).Sprintf(endMsg))

	if c.successImagesFile != "" {
		file, err := os.OpenFile(c.successImagesFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
		if err != nil {
			return fmt.Errorf("open file %v error: %v", c.successImagesFile, err)
		}

		if c.outputImagesFormat == "json" {
			encoder := json.NewEncoder(file)
			if err := encoder.Encode(c.successImagesList.Content()); err != nil {
				return fmt.Errorf("marshal success images error: %v", err)
			}
		} else {
			encoder := yaml.NewEncoder(file)
			if err := encoder.Encode(c.successImagesList.Content()); err != nil {
				return fmt.Errorf("marshal success images error: %v", err)
			}
		}
	}

	_, failedSyncTaskCountTotal := c.failedTaskCounter.Value()
	if failedSyncTaskCountTotal != 0 {
		return fmt.Errorf("failed tasks exist")
	}
	return nil
}

func (c *Client) openRoutinesHandleTaskAndWaitForFinish() {
	broadcastChan := concurrent.NewBroadcastChan(c.routineNum)
	broadcastChan.Broadcast()

	go func() {
		for {
			// if all the worker routines is hung and taskList is empty, stop everything
			<-broadcastChan.TotalHungChan()
			if c.taskList.Len() == 0 {
				broadcastChan.Close()
			}
		}
	}()

	concurrent.CreateRoutinesAndWaitForFinish(c.routineNum, func() {
		for {
			closed := broadcastChan.Wait()

			// run out of exist tasks
			for {
				item := c.taskList.PopFront()
				// no more tasks need to handle
				if item == nil {
					break
				}

				tTask := item.(task.Task)

				c.logger.Infof("Executing %v...", tTask.String())
				nextTasks, message, err := tTask.Run()

				count, total := c.taskCounter.Increase()
				finishedNumString := color.New(color.FgGreen).Sprintf("%d", count)
				totalNumString := color.New(color.FgGreen).Sprintf("%d", total)

				if err != nil {
					c.failedTaskList.PushBack(tTask)
					c.failedTaskCounter.IncreaseTotal()
					c.logger.Errorf("Failed to executed %v: %v. Now %v/%v tasks have been processed.", tTask.String(), err,
						finishedNumString, totalNumString)
				} else {
					if tTask.Type() == task.ManifestType {
						// TODO: the ignored images will not be recorded in success images list
						c.successImagesList.Add(tTask.GetSource().String(), tTask.GetDestination().String())
					}

					if len(message) != 0 {
						c.logger.Infof("Finish %v: %v. Now %v/%v tasks have been processed.", tTask.String(), message,
							finishedNumString, totalNumString)
					} else {
						c.logger.Infof("Finish %v. Now %v/%v tasks have been processed.", tTask.String(),
							finishedNumString, totalNumString)
					}
				}

				if nextTasks != nil {
					for _, t := range nextTasks {
						c.taskList.PushFront(t)
						c.taskCounter.IncreaseTotal()
					}
					broadcastChan.Broadcast()
				}
			}

			if closed {
				return
			}
		}
	})
}
