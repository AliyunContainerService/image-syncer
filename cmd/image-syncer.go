package cmd

import (
	"fmt"
	"os"

	"github.com/AliyunContainerService/image-syncer/pkg/client"
	"github.com/AliyunContainerService/image-syncer/pkg/utils"

	"github.com/spf13/cobra"
)

var (
	logPath, configFile, authFile, imagesFile, successImagesFile, outputImagesFormat string

	procNum, retries int

	osFilterList, archFilterList []string

	forceUpdate, version bool
)

// RootCmd describes "image-syncer" command
var RootCmd = &cobra.Command{
	Use:     "image-syncer",
	Aliases: []string{"image-syncer"},
	Short:   "A docker registry image synchronization tool",
	Long: `A Fast and Flexible docker registry image synchronization tool implement by Go. 
	
	Complete documentation is available at https://github.com/AliyunContainerService/image-syncer`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceErrors = true

		if version {
			fmt.Println("app-version: ", utils.Version)
			fmt.Println("git-branch: ", utils.Branch)
			fmt.Println("git-commit: ", utils.Commit)
			fmt.Println("build-timestamp: ", utils.Timestamp)
			return nil
		}

		// work starts here
		client, err := client.NewSyncClient(configFile, authFile, imagesFile, logPath, successImagesFile, outputImagesFormat,
			procNum, retries, utils.RemoveEmptyItems(osFilterList), utils.RemoveEmptyItems(archFilterList), forceUpdate)
		if err != nil {
			return fmt.Errorf("init sync client error: %v", err)
		}

		cmd.SilenceUsage = true
		return client.Run()
	},
}

func init() {
	RootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file path. This flag is deprecated and will be removed in the future. Please use --auth and --images instead.")
	RootCmd.PersistentFlags().StringVar(&authFile, "auth", "", "auth file path. This flag need to be pair used with --images.")
	RootCmd.PersistentFlags().StringVar(&imagesFile, "images", "", "images file path. This flag need to be pair used with --auth")
	RootCmd.PersistentFlags().StringVar(&logPath, "log", "", "log file path (default in os.Stderr)")
	RootCmd.PersistentFlags().IntVarP(&procNum, "proc", "p", 5, "numbers of working goroutines")
	RootCmd.PersistentFlags().IntVarP(&retries, "retries", "r", 2, "times to retry failed task")
	RootCmd.PersistentFlags().StringArrayVar(&osFilterList, "os", []string{}, "os list to filter source tags, not works for docker v2 schema1 and OCI media")
	RootCmd.PersistentFlags().StringArrayVar(&archFilterList, "arch", []string{}, "architecture list to filter source tags, not works for OCI media")
	RootCmd.PersistentFlags().BoolVar(&forceUpdate, "force", false, "force update manifest whether the destination manifest exists")
	RootCmd.PersistentFlags().StringVar(&successImagesFile, "output-success-images", "", "output success images in a new file")
	RootCmd.PersistentFlags().StringVar(&outputImagesFormat, "output-images-format", "yaml", "success images output format, json or yaml")
	RootCmd.PersistentFlags().BoolVar(&version, "version", false, "print app version")
}

// Execute executes the RootCmd
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
