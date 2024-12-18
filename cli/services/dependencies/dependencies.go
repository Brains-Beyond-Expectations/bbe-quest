package dependencies

import (
	"os/exec"

	"github.com/nicolajv/bbe-quest/services/logger"
)

func VerifyDependencies() bool {
	dependencyChecks := []string{
		"talosctl",
		"nmap --version",
		"yq --version",
		"grep --version",
		"aws --version",
		"bash --version",
	}

	errors := 0
	for _, check := range dependencyChecks {
		cmd := exec.Command("bash", "-c", check)
		err := cmd.Run()
		if err != nil {
			errors++
			logger.Error("Dependency check failed", err)
		}
	}

	return errors == 0
}
