package kubernetes_service

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/command"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
)

var execCommand = exec.Command
var portForwardTimeout = 30 * time.Second

type KubernetesService struct{}

var forwardingPort = regexp.MustCompile(`Forwarding from 127\.0\.0\.1:(\d+)`)

// Forwards a free local port to a service, returning the port and a function that stops forwarding
func (kubernetesService KubernetesService) PortForward(context string, namespace string, service string, port int) (int, func(), error) {
	cmd := execCommand("kubectl", "port-forward", "service/"+service, fmt.Sprintf(":%d", port),
		"--namespace", namespace, "--context", context)
	logger.Debug(fmt.Sprintf("Command: %s", cmd.String()))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, nil, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return 0, nil, kubectlError(err)
	}
	stop := func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}

	found := make(chan int, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if match := forwardingPort.FindStringSubmatch(scanner.Text()); match != nil {
				localPort, _ := strconv.Atoi(match[1])
				found <- localPort
				return
			}
		}
		close(found)
	}()

	select {
	case localPort, ok := <-found:
		if ok {
			return localPort, stop, nil
		}
	case <-time.After(portForwardTimeout):
	}

	stop()
	return 0, nil, fmt.Errorf("Failed to forward a port to service `%s`: %s", service, strings.TrimSpace(stderr.String()))
}

func (kubernetesService KubernetesService) GetSecretValue(context string, namespace string, secret string, key string) (string, error) {
	data, found, err := getSecretData(context, namespace, secret)
	if err != nil {
		return "", err
	}
	value, hasKey := data[key]
	if !found || !hasKey {
		return "", fmt.Errorf("Secret `%s` has no key `%s`", secret, key)
	}

	return value, nil
}

// Adds the values to the secret, creating it when it doesn't exist, and keeps its other keys
func (kubernetesService KubernetesService) SetSecretValues(context string, namespace string, secret string, values map[string]string) error {
	data, found, err := getSecretData(context, namespace, secret)
	if err != nil {
		return err
	}
	for key, value := range values {
		data[key] = value
	}

	return writeSecret(context, namespace, secret, data, found)
}

func (kubernetesService KubernetesService) RemoveSecretValues(context string, namespace string, secret string, keys []string) error {
	data, found, err := getSecretData(context, namespace, secret)
	if err != nil || !found {
		return err
	}
	for _, key := range keys {
		delete(data, key)
	}

	return writeSecret(context, namespace, secret, data, true)
}

// The secret's decoded values, and whether it exists
func getSecretData(context string, namespace string, secret string) (map[string]string, bool, error) {
	response, err := kubectl(context, nil, "get", "secret", secret, "--namespace", namespace, "--output", "json")
	if err != nil {
		if strings.Contains(string(response), "NotFound") {
			return map[string]string{}, false, nil
		}
		return nil, false, fmt.Errorf("Failed to read secret `%s`: %w", secret, err)
	}

	var manifest struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(response, &manifest); err != nil {
		return nil, false, fmt.Errorf("Failed to read secret `%s`: %w", secret, err)
	}

	data := map[string]string{}
	for key, encoded := range manifest.Data {
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, false, fmt.Errorf("Failed to read key `%s` of secret `%s`: %w", key, secret, err)
		}
		data[key] = string(decoded)
	}

	return data, true, nil
}

// Replaces the whole secret, so removed keys go too
func writeSecret(context string, namespace string, secret string, data map[string]string, exists bool) error {
	encoded := map[string]string{}
	for key, value := range data {
		encoded[key] = base64.StdEncoding.EncodeToString([]byte(value))
	}

	manifest, err := json.Marshal(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"type":       "Opaque",
		"metadata": map[string]interface{}{
			"name":      secret,
			"namespace": namespace,
			"labels":    map[string]string{"app.kubernetes.io/managed-by": "bbe"},
		},
		"data": encoded,
	})
	if err != nil {
		return err
	}

	verb := "create"
	if exists {
		verb = "replace"
	}
	if _, err := kubectl(context, manifest, verb, "--filename", "-"); err != nil {
		return fmt.Errorf("Failed to save secret `%s`: %w", secret, err)
	}

	return nil
}

func kubectl(context string, stdin []byte, args ...string) ([]byte, error) {
	cmd := execCommand("kubectl", append(args, "--context", context)...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	// Secrets are read and written here, so only the command is logged
	logger.Debug(fmt.Sprintf("Command: %s", cmd.String()))

	response, err := command.Output(cmd)
	if err != nil {
		return response, fmt.Errorf("%w%s", kubectlError(err), output(response))
	}

	return response, nil
}

func kubectlError(err error) error {
	if errors.Is(err, exec.ErrNotFound) {
		return errors.New("kubectl is needed to configure this package, please install it")
	}
	return err
}

func output(response []byte) string {
	if trimmed := strings.TrimSpace(string(response)); trimmed != "" {
		return "\n" + trimmed
	}
	return ""
}
