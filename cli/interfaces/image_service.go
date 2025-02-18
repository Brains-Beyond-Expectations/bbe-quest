package interfaces

import "github.com/Brains-Beyond-Expectations/bbe-quest/models"

type ImageServiceInterface interface {
	CreateImage(nodeType models.NodeType, outputDir string) (string, error)
}
