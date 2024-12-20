package cmd

import (
	"github.com/Brains-Beyond-Expectations/bbe-quest/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/services/logger"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Show bbe version",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		logger.Debug("Showing version")
		logger.Infof("Version: %s", constants.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
