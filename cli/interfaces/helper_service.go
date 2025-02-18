package interfaces

import (
	"os/exec"
	"time"
)

type HelperServiceInterface interface {
	CheckIfFileExists(file string) (*time.Time, bool)
	PipeCommands(commands ...*exec.Cmd) ([]byte, error)
	DeleteEmptyStrings(s []string) []string
	IsWsl() bool
	IsValidIp(ip string) bool
	GetConfigDir() string
	GetConfigFilePath(name string) string
}
