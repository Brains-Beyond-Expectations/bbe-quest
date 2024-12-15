package dependencies

import (
	"os/exec"

	"github.com/sirupsen/logrus"
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
			logrus.Errorf("Dependency check failed for %s, please install it", check)
		}
	}

	return errors == 0
}
