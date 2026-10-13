package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_LocalPath_OfAChartFolder(t *testing.T) {
	path, local := ChartEntry{RepositoryUrl: "file:///charts/bbe-networking"}.LocalPath()

	assert.True(t, local)
	assert.Equal(t, "/charts/bbe-networking", path)
}

func Test_LocalPath_OfAChartRepository(t *testing.T) {
	_, local := ChartEntry{RepositoryUrl: "https://brains-beyond-expectations.github.io/bbe-charts"}.LocalPath()

	assert.False(t, local)
}
