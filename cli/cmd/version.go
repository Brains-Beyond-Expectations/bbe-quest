package cmd

import (
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Show bbe version",
	Args:    cobra.ExactArgs(0),
	Run:     versionCommand,
}

func versionCommand(cmd *cobra.Command, args []string) {
	logger.Debug("Showing version")
	logger.Infof("Version: %s", constants.Version)
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
