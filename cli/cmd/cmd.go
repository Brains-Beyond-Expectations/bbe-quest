package cmd

import (
	"os"

	"github.com/nicolajv/bbe-quest/services/logger"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bbe",
	Short: "a cli for managing your Talos k8s cluster",
	Long:  `bbe is a cli for managing your Talos k8s cluster.`,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(0)
	}

	// No command was provided
	if len(os.Args) == 1 {
		err := rootCmd.Help()
		if err != nil {
			logger.Error("Error while printing help message", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}
