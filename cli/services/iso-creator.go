package services

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/nicolajv/bbe-quest/helper"
	"github.com/sirupsen/logrus"
)

func CheckIfIsoExists(outputDir string) bool {
	_, err := exec.LookPath(fmt.Sprintf("%s/metal-amd64.iso", outputDir))
	return err == nil
}

func CreateIso(outputDir string, extensions []string) (string, error) {
	extensions, err := getExtensionImages(extensions)
	if err != nil {
		return "", err
	}

	extensionsStrings := []string{}
	for _, extension := range extensions {
		extensionsStrings = append(extensionsStrings, "--system-extension-image", extension)
	}

	_, err = runDockerCommand(outputDir, extensionsStrings)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/metal-amd64.iso", outputDir), nil
}

func getExtensionImages(extensionIdentifiers []string) ([]string, error) {
	cmd1 := exec.Command("crane", "export", "ghcr.io/siderolabs/extensions:v1.8.0")
	cmd2 := exec.Command("tar", "x", "-O", "image-digests")
	cmd3 := exec.Command("grep", "-E", ".*("+strings.Join(extensionIdentifiers, ":|")+")")

	output, err := helper.PipeCommands(cmd1, cmd2, cmd3)
	if err != nil {
		logrus.WithFields(logrus.Fields{"error": err, "output": string(output)}).Error("Error while getting extension images")
		return nil, err
	}

	images := strings.Split(string(output), "\n")

	images = helper.DeleteEmptyStrings(images)

	return images, nil
}

func runDockerCommand(outputDir string, extensionsStrings []string) (string, error) {
	args := []string{"run", "--rm", "-t", "-v", fmt.Sprintf("%s:/out", outputDir), "ghcr.io/siderolabs/imager:v1.8.0", "iso"}
	args = append(args, extensionsStrings...)

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.WithFields(logrus.Fields{"error": err, "output": string(output)}).Error("Error while generating ISO")
		return "", err
	}
	return string(output), nil

}
