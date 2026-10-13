package cmd

import (
	"os"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bbe",
	Short: "a cli for managing your Talos k8s cluster",
	Long:  `bbe is a cli for managing your Talos k8s cluster.`,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&constants.BbeLibraryUrl, "library", defaultLibrary(),
		"a URL or file to read the package library from, to test charts that aren't released yet (or set BBE_LIBRARY)")
}

func defaultLibrary() string {
	if library := os.Getenv("BBE_LIBRARY"); library != "" {
		return library
	}

	return constants.DefaultBbeLibraryUrl
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
