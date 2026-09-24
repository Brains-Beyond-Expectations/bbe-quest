package image_service

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
)

var httpGet = http.Get
var osCreate = os.Create
var ioCopy = io.Copy

type ImageService struct{}

// talosImageVersion is the Talos version baked into the node images below. It must be
// bumped alongside the schematic image links, and is also used both to warn when the
// local talosctl client has drifted from the version the node image was built for, and
// to pin `talosctl gen config` to the same schema version the node image was built for
// (see TalosService.GenerateConfig).
const talosImageVersion = "v1.14.1"

var IntelNuc = models.NodeType{
	OutputFile:   "metal-amd64.iso",
	ImagerType:   "iso",
	ImageLink:    fmt.Sprintf("https://factory.talos.dev/image/434a0300db532066f1098e05ac068159371d00f0aba0a3103a0e826e83825c82/%s/metal-amd64.iso", talosImageVersion),
	Extensions:   []string{"intel-ucode", "gvisor", "iscsi-tools", "util-linux-tools"},
	TalosVersion: talosImageVersion,
}

var RaspberryPi = models.NodeType{
	OutputFile:   "metal-arm64.raw.xz",
	ImagerType:   "rpi_generic",
	ImageLink:    fmt.Sprintf("https://factory.talos.dev/image/f8a903f101ce10f686476024898734bb6b36353cc4d41f348514db9004ec0a9d/%s/metal-arm64.raw.xz", talosImageVersion),
	Extensions:   []string{"iscsi-tools", "util-linux-tools"},
	TalosVersion: talosImageVersion,
}

func (imageService ImageService) CreateImage(nodeType models.NodeType, outputDir string) (string, error) {
	return imageService.downloadImage(nodeType, outputDir)
}

func (imageService ImageService) downloadImage(nodeType models.NodeType, outputDir string) (string, error) {
	outputFile := fmt.Sprintf("%s/%s", outputDir, nodeType.OutputFile)
	resp, err := httpGet(nodeType.ImageLink)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	logger.Debug("Creating directory...")
	dirCreationErr := os.MkdirAll(outputDir, os.ModePerm)
	if dirCreationErr != nil {
		return "", dirCreationErr
	}
	logger.Debug("Directory created")

	out, err := osCreate(outputFile)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = ioCopy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return outputFile, nil
}
