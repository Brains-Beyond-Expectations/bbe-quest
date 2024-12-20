package imagecreator

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/Brains-Beyond-Expectations/bbe-quest/helper"
	"github.com/Brains-Beyond-Expectations/bbe-quest/services/logger"
)

type NodeType struct {
	OutputFile          string
	ImagerType          string
	ImagerArch          string
	Extensions          []string
	GetOverlayImageArgs func() ([]string, error)
}

var IntelNuc = NodeType{
	OutputFile: "metal-amd64.iso",
	ImagerType: "iso",
	ImagerArch: "amd64",
	Extensions: []string{"intel-ucode", "gvisor", "iscsi-tools"},
	GetOverlayImageArgs: func() ([]string, error) {
		return []string{}, nil
	},
}

var RaspberryPi = NodeType{
	OutputFile:          "metal-arm64.raw.xz",
	ImagerType:          "rpi_generic",
	ImagerArch:          "arm64",
	Extensions:          []string{"iscsi-tools"},
	GetOverlayImageArgs: getRaspberryPiOverlayImages,
}

func CreateImage(nodeType NodeType, outputDir string) (string, error) {
	extensions, err := getExtensionImages(nodeType.Extensions)
	if err != nil {
		return "", err
	}

	extensionsStrings := []string{}
	for _, extension := range extensions {
		extensionsStrings = append(extensionsStrings, "--system-extension-image", extension)
	}

	_, err = runDockerCommand(nodeType, outputDir, extensionsStrings)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/metal-amd64.iso", outputDir), nil
}

func getExtensionImages(extensionIdentifiers []string) ([]string, error) {
	cmd1 := exec.Command("crane", "export", "ghcr.io/siderolabs/extensions:v1.9.0")
	cmd2 := exec.Command("tar", "x", "-O", "image-digests")
	cmd3 := exec.Command("grep", "-E", ".*("+strings.Join(extensionIdentifiers, ":|")+")")

	output, err := helper.PipeCommands(cmd1, cmd2, cmd3)
	if err != nil {
		logger.Error("Error while getting extension images", err)
		return nil, err
	}

	images := strings.Split(string(output), "\n")

	images = helper.DeleteEmptyStrings(images)

	return images, nil
}

func runDockerCommand(nodeType NodeType, outputDir string, extensionsStrings []string) (string, error) {
	args := []string{"run", "--rm", "-t", "-v", fmt.Sprintf("%s:/out", outputDir), "--privileged", "ghcr.io/siderolabs/imager:v1.9.0", nodeType.ImagerType, "--arch", nodeType.ImagerArch}
	args = append(args, extensionsStrings...)

	overlayImages, err := nodeType.GetOverlayImageArgs()
	if err != nil {
		return "", err
	}

	args = append(args, overlayImages...)

	// Print args as one space separated string
	logger.Debug(strings.Join(args, " "))

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Debug(string(output))
		logger.Error("Error while generating image", nil)
		return "", err
	}
	return string(output), nil
}

func getRaspberryPiOverlayImages() ([]string, error) {
	cmd1 := exec.Command("crane", "export", "ghcr.io/siderolabs/overlays:v1.9.0")
	cmd2 := exec.Command("tar", "x", "-O", "overlays.yaml")
	cmd3 := exec.Command("yq", `.overlays[] | select(.name=="rpi_generic") | "--overlay-image " + .image + "@" + .digest + " --overlay-name=" + .name`)

	output, err := helper.PipeCommands(cmd1, cmd2, cmd3)
	if err != nil {
		logger.Error("Error while getting overlay images", err)
		return nil, err
	}
	return strings.Split(string(output), " "), nil
}
