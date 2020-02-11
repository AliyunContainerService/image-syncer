package cmd

import (
	"fmt"
	"os"

	"github.com/AliyunContainerService/image-syncer/pkg/client"
	"github.com/spf13/cobra"
)

var (
	logPath, configFile, recordPath, defaultRegistry, defaultNamespace string

	procNum, retries int
)

// RootCmd describes "image-syncer" command
var RootCmd = &cobra.Command{
	Use:     "image-syncer",
	Aliases: []string{"image-syncer"},
	Short:   "A docker registry image synchronization tool",
	Long: `A Fast and Flexible docker registry image synchronization tool implement by Go. 
	
	Complete documentation is available at https://github.com/AliyunContainerService/image-syncer`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// work starts here
		client, err := client.NewSyncClient(configFile, logPath, recordPath, procNum, retries, defaultRegistry, defaultNamespace)
		if err != nil {
			return fmt.Errorf("init sync client error: %v", err)
		}

		client.Run()
		return nil
	},
}

func init() {
	var defaultLogPath, defaultConfigFile, defaultRecordPath string

	pwd, err := os.Getwd()
	if err == nil {
		defaultLogPath = ""
		defaultConfigFile = pwd + "/" + "image-syncer.json"
		defaultRecordPath = pwd + "/" + "records"
	}

	RootCmd.PersistentFlags().StringVar(&configFile, "config", defaultConfigFile, "config file path")
	RootCmd.PersistentFlags().StringVar(&logPath, "log", defaultLogPath, "log file path (default in os.Stderr)")
	RootCmd.PersistentFlags().StringVar(&recordPath, "records", defaultRecordPath,
		"records file path, to record the blobs that have been synced, auto generated if not exist")
	RootCmd.PersistentFlags().StringVar(&defaultRegistry, "registry", os.Getenv("DEFAULT_REGISTRY"),
		"default destinate registry url when destinate registry is not given in the config file, can also be set with DEFAULT_REGISTRY environment value")
	RootCmd.PersistentFlags().StringVar(&defaultNamespace, "namespace", os.Getenv("DEFAULT_NAMESPACE"),
		"default destinate namespace when destinate namespace is not given in the config file, can also be set with DEFAULT_NAMESPACE environment value")
	RootCmd.PersistentFlags().IntVarP(&procNum, "proc", "p", 5, "numbers of working goroutines")
	RootCmd.PersistentFlags().IntVarP(&retries, "retries", "r", 2, "times to retry failed task")
}

// Execute executes the RootCmd
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
