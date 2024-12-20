package dependencies

import (
	"os/exec"

	"github.com/Brains-Beyond-Expectations/bbe-quest/services/logger"
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
			logger.Error("Dependency check failed", nil)
		}
	}

	dockerRunninng := exec.Command("docker", "info")
	err := dockerRunninng.Run()
	if err != nil {
		errors++
		logger.Error("Please make sure Docker is running", nil)
	}

	return errors == 0
}
