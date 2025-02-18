package dependency_service

import (
	"fmt"
	"os/exec"

	"github.com/Brains-Beyond-Expectations/bbe-quest/misc/logger"
)

type DependencyService struct{}

func (dependencyService DependencyService) VerifyDependencies() bool {
	dependencyChecks := map[string]struct {
		args []string
	}{
		"talosctl": {args: []string{""}},
		"nmap":     {args: []string{"--version"}},
		"grep":     {args: []string{"--version"}},
		"bash":     {args: []string{"--version"}},
		"awk":      {args: []string{"--version"}},
	}

	errors := 0
	for dependency, check := range dependencyChecks {
		cmd := exec.Command("bash", "-c", buildCommand(dependency, check.args))
		err := cmd.Run()
		if err != nil {
			errors++
			logger.Error(fmt.Sprintf("Dependency %s is not installed, please install it.", dependency), nil)
		}
	}

	return errors == 0
}

var buildCommand = func(dependency string, args []string) string {
	command := dependency
	for _, arg := range args {
		command += " " + arg
	}
	return command
}
