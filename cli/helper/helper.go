package helper

import (
	"os/exec"
	"regexp"
	"strings"
)

func CheckIfFileExists(file string) bool {
	_, err := exec.LookPath(file)
	return err == nil
}

func PipeCommands(commands ...*exec.Cmd) ([]byte, error) {
	for i := 0; i < len(commands)-1; i++ {
		stdout, err := commands[i].StdoutPipe()
		if err != nil {
			return commands[i].Output()
		}
		commands[i+1].Stdin = stdout
		if err := commands[i].Start(); err != nil {
			return nil, err
		}
	}
	return commands[len(commands)-1].Output()
}

func DeleteEmptyStrings(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func IsWsl() bool {
	cmd := exec.Command("uname", "-a")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "microsoft") && strings.Contains(string(output), "WSL")
}

func IsValidIp(ip string) bool {
	pattern := `^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.?\b){4}$`
	match, err := regexp.MatchString(pattern, ip)
	return err == nil && match
}
