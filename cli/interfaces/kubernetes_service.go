package interfaces

type KubernetesServiceInterface interface {
	PortForward(context string, namespace string, service string, port int) (int, func(), error)
	GetSecretValue(context string, namespace string, secret string, key string) (string, error)
	SetSecretValues(context string, namespace string, secret string, values map[string]string) error
	RemoveSecretValues(context string, namespace string, secret string, keys []string) error
}
