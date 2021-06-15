package cmd

import (
	"fmt"
	"os"

	"github.com/AliyunContainerService/image-syncer/pkg/client"
	"github.com/spf13/cobra"
)

var (
	logPath, configFile, authFile, imageFile, platformFile, defaultRegistry, defaultNamespace string

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
		client, err := client.NewSyncClient(configFile, authFile, imageFile, platformFile, logPath, procNum, retries, defaultRegistry, defaultNamespace)
		if err != nil {
			return fmt.Errorf("init sync client error: %v", err)
		}

		client.Run()
		return nil
	},
}

func init() {

	RootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file path. This flag is deprecated and will be removed in the future. Please use --auth and --images instead.")
	RootCmd.PersistentFlags().StringVar(&authFile, "auth", "", "auth file path. This flag need to be pair used with --images.")
	RootCmd.PersistentFlags().StringVar(&imageFile, "images", "", "images file path. This flag need to be pair used with --auth")
	RootCmd.PersistentFlags().StringVar(&platformFile, "platform", "", "platform file path. This flag is optional")
	RootCmd.PersistentFlags().StringVar(&logPath, "log", "", "log file path (default in os.Stderr)")
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
