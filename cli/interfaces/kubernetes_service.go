package interfaces

import "github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"

type KubernetesServiceInterface interface {
	PortForward(context string, namespace string, service string, port int) (int, func(), error)
	GetSecretValue(context string, namespace string, secret string, key string) (string, error)
	SetSecretValues(context string, namespace string, secret string, values map[string]string) error
	RemoveSecretValues(context string, namespace string, secret string, keys []string) error
	NamespaceVolumes(context string, namespace string) ([]models.NamespaceVolume, bool, error)
	Patch(context string, namespace string, resource string, name string, patch string) error
	DeleteNamespace(context string, namespace string) error
}
