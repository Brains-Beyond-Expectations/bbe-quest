package helper

import (
	"os/exec"
)

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
