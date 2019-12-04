package main

import (
	"fmt"
	"github.com/AliyunContainerService/image-syncer/cmd"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const (
	intervalEnv = "DEFAULT_SYNC_INTERVAL"
)

func main() {
	intervalStr, found := os.LookupEnv(intervalEnv)
	if !found || intervalStr == "" {
		cmd.Execute()
		return
	}

	interval, err := strconv.Atoi(intervalStr)
	if err != nil {
		panic(fmt.Errorf("invalid envar DEFAULT_SYNC_INTERVAL: %v", err))
	}
	intervalDuration := time.Duration(interval) * time.Second
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill, syscall.SIGTERM)
	timer := time.NewTimer(intervalDuration)
	defer timer.Stop()

	for {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(intervalDuration)
		select {
		case <-quit:
			return
		case <-timer.C:
			cmd.Execute()
			continue
		}
	}
}
