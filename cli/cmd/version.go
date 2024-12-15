package cmd

import (
	"fmt"

	"github.com/nicolajv/bbe-quest/constants"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"rev"},
	Short:   "Show bbe version",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(constants.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
