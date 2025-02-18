package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func Test_versionCommand_Succeeds_WithoutArgs(t *testing.T) {
	cmd := &cobra.Command{}
	args := []string{}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code did panic")
		}
	}()
	versionCommand(cmd, args)
}

func Test_versionCommand_Succeeds_WithArgs(t *testing.T) {
	cmd := &cobra.Command{}
	args := []string{"unexpectedArg"}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code did panic")
		}
	}()
	versionCommand(cmd, args)
}
