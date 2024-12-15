package cmd

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bbe",
	Short: "bbe - a simple CLI to transform and inspect strings",
	Long: `bbe is a super fancy CLI (kidding)
      
   One can use bbe to modify or inspect strings straight from the terminal`,
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
			logrus.Error("Error while printing help message")
			os.Exit(1)
		}
		os.Exit(0)
	}
}
