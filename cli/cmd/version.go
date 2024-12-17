package cmd

import (
	"fmt"

	"github.com/nicolajv/bbe-quest/constants"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Show bbe version",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Info(fmt.Sprintf("Version: %s", constants.Version))
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
