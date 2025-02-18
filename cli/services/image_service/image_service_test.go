package image_service

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CreateImage_Succeeds_WhenDownloadSucceeds(t *testing.T) {
	imageService := ImageService{}
	httpGet = func(url string) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("fake content")),
		}, nil
	}
	osCreate = func(name string) (*os.File, error) {
		return os.CreateTemp("", "testfile")
	}

	outputFile, err := imageService.CreateImage(IntelNuc, "/tmp")
	assert.NotEmpty(t, outputFile)
	assert.Nil(t, err)
}

func Test_CreateImage_Fails_WhenDownloadFiles(t *testing.T) {
	imageService := ImageService{}
	httpGet = func(url string) (*http.Response, error) {
		return nil, errors.New("failed to download")
	}
	osCreate = func(name string) (*os.File, error) {
		return os.CreateTemp("", "testfile")
	}

	outputFile, err := imageService.CreateImage(RaspberryPi, "/tmp")
	assert.Empty(t, outputFile)
	assert.NotNil(t, err)
}

func Test_CreateImage_Fails_WhenFileCanNotBeCreated(t *testing.T) {
	imageService := ImageService{}
	httpGet = func(url string) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("fake content")),
		}, nil
	}
	osCreate = func(name string) (*os.File, error) {
		return nil, errors.New("failed to create file")
	}

	outputFile, err := imageService.CreateImage(IntelNuc, "/tmp")
	assert.Empty(t, outputFile)
	assert.NotNil(t, err)
}

func Test_CreateImage_Fails_WhenFileContentsCanNotBeCopied(t *testing.T) {
	imageService := ImageService{}
	httpGet = func(_ string) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("fake content")),
		}, nil
	}
	osCreate = func(_ string) (*os.File, error) {
		return os.CreateTemp("", "testfile")
	}
	ioCopy = func(_ io.Writer, _ io.Reader) (int64, error) {
		return 0, errors.New("failed to copy file contents")
	}

	outputFile, err := imageService.CreateImage(IntelNuc, "/tmp")
	assert.Empty(t, outputFile)
	assert.NotNil(t, err)
}
