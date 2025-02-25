package helper_service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
)

var configDir string = ".bbe"
var execCommand = exec.Command

type HelperService struct{}

func (helperService HelperService) CheckIfFileExists(file string) (*time.Time, bool) {
	fileInfo, err := os.Stat(file)
	if err != nil {
		return nil, false
	}

	modTime := fileInfo.ModTime()

	return &modTime, err == nil
}

func (helperService HelperService) PipeCommands(commands ...*exec.Cmd) ([]byte, error) {
	if len(commands) == 0 {
		return nil, fmt.Errorf("No commands provided")
	}

	if len(commands) == 1 {
		return commands[0].CombinedOutput()
	}

	for i := 0; i < len(commands)-1; i++ {
		stdout, err := commands[i].StdoutPipe()
		if err != nil {
			logger.Debug(fmt.Sprintf("Failed to create pipe: %v", err))
			return nil, err
		}
		commands[i+1].Stdin = stdout
	}

	for i := 0; i < len(commands)-1; i++ {
		if err := commands[i].Start(); err != nil {
			logger.Debug(fmt.Sprintf("Failed to start command %d: %v", i, err))
			return nil, err
		}
	}

	output, err := commands[len(commands)-1].CombinedOutput()
	if err != nil {
		logger.Debug(fmt.Sprintf("Last command failed: %v\nOutput: %s", err, output))
		return output, err
	}

	for i := 0; i < len(commands)-1; i++ {
		if err := commands[i].Wait(); err != nil {
			logger.Debug(fmt.Sprintf("Command %d failed while waiting: %v", i, err))
			return output, err
		}
	}

	return output, nil
}

func (helperService HelperService) DeleteEmptyStrings(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func (helperService HelperService) IsWsl() bool {
	cmd := execCommand("uname", "-a")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Debug(string(output))
		return false
	}

	return strings.Contains(string(output), "microsoft") && strings.Contains(string(output), "WSL")
}

func (helperService HelperService) IsValidIp(ip string) bool {
	pattern := `^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.?\b){4}$`
	match, err := regexp.MatchString(pattern, ip)
	return err == nil && match
}

func (helperService HelperService) GetConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return filepath.Join(homeDir, configDir)
}

func (helperService HelperService) GetConfigFilePath(name string) string {
	configDir := helperService.GetConfigDir()
	return fmt.Sprintf("%s/%s", configDir, name)
}
