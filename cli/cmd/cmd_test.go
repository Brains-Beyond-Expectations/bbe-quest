package cmd

import (
	"testing"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/stretchr/testify/assert"
)

func Test_defaultLibrary_IsTheReleasedLibrary(t *testing.T) {
	t.Setenv("BBE_LIBRARY", "")

	assert.Equal(t, constants.DefaultBbeLibraryUrl, defaultLibrary())
}

func Test_defaultLibrary_CanBeSetFromTheEnvironment(t *testing.T) {
	t.Setenv("BBE_LIBRARY", "./library.yaml")

	assert.Equal(t, "./library.yaml", defaultLibrary())
}

func Test_rootCmd_HasALibraryFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("library")

	assert.NotNil(t, flag)
}
