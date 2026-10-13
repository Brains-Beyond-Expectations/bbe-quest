package kubernetes_service

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
)

type fakeCall struct {
	args      string
	stdinFile string
}

// What the call was given on stdin, once it has run
func (call fakeCall) stdin() string {
	content, _ := os.ReadFile(call.stdinFile)
	return string(content)
}

// Answers kubectl by the start of its arguments, recording each call and saving what it was given on stdin
func fakeKubectl(t *testing.T, answers map[string]string) *[]fakeCall {
	calls := &[]fakeCall{}
	dir := t.TempDir()
	execCommand = func(_ string, args ...string) *exec.Cmd {
		joined := strings.Join(args, " ")
		stdinFile := filepath.Join(dir, fmt.Sprintf("stdin-%d", len(*calls)))
		*calls = append(*calls, fakeCall{args: joined, stdinFile: stdinFile})
		for prefix, script := range answers {
			if strings.HasPrefix(joined, prefix) {
				return exec.Command("sh", "-c", `cat > "$0"; `+script, stdinFile)
			}
		}
		t.Fatalf("unexpected command: %s", joined)
		return nil
	}
	t.Cleanup(func() { execCommand = exec.Command })

	return calls
}

const secretJson = `{"data": {"prowlarr": "a2V5", "beyondhd.apiKey": "b2xk"}}`

func Test_PortForward_Succeeds(t *testing.T) {
	var args []string
	execCommand = func(_ string, received ...string) *exec.Cmd {
		args = received
		return exec.Command("sh", "-c", "echo 'Forwarding from 127.0.0.1:43210 -> 9696'; echo 'Forwarding from [::1]:43210 -> 9696'; exec sleep 30")
	}
	defer func() { execCommand = exec.Command }()

	port, stop, err := KubernetesService{}.PortForward("context", "bbe-media", "bbe-media-prowlarr", 9696)

	assert.NoError(t, err)
	assert.Equal(t, 43210, port)
	assert.Equal(t, []string{"port-forward", "service/bbe-media-prowlarr", ":9696", "--namespace", "bbe-media", "--context", "context"}, args)
	stop()
}

func Test_PortForward_Fails_WhenKubectlExits(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("sh", "-c", `echo 'error: services "bbe-media-prowlarr" not found' >&2; exit 1`)
	}
	defer func() { execCommand = exec.Command }()

	port, stop, err := KubernetesService{}.PortForward("context", "bbe-media", "bbe-media-prowlarr", 9696)

	assert.Zero(t, port)
	assert.Nil(t, stop)
	assert.EqualError(t, err, "Failed to forward a port to service `bbe-media-prowlarr`: error: services \"bbe-media-prowlarr\" not found")
}

func Test_PortForward_Fails_WhenItNeverForwards(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("sleep", "30")
	}
	portForwardTimeout = 100 * time.Millisecond
	defer func() {
		execCommand = exec.Command
		portForwardTimeout = 30 * time.Second
	}()

	_, _, err := KubernetesService{}.PortForward("context", "bbe-media", "bbe-media-prowlarr", 9696)

	assert.ErrorContains(t, err, "Failed to forward a port to service `bbe-media-prowlarr`")
}

func Test_PortForward_Fails_WithoutKubectl(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("bbe-missing-kubectl")
	}
	defer func() { execCommand = exec.Command }()

	_, _, err := KubernetesService{}.PortForward("context", "bbe-media", "bbe-media-prowlarr", 9696)

	assert.EqualError(t, err, "kubectl is needed to configure this package, please install it")
}

func Test_GetSecretValue_Succeeds(t *testing.T) {
	calls := fakeKubectl(t, map[string]string{"get secret": "printf '%s' '" + secretJson + "'"})

	value, err := KubernetesService{}.GetSecretValue("context", "bbe-media", "bbe-media-api-keys", "prowlarr")

	assert.NoError(t, err)
	assert.Equal(t, "key", value)
	assert.Equal(t, "get secret bbe-media-api-keys --namespace bbe-media --output json --context context", (*calls)[0].args)
}

func Test_GetSecretValue_Fails(t *testing.T) {
	cases := map[string]string{
		"printf '%s' '" + secretJson + "'":                                   "Secret `bbe-media-api-keys` has no key `sonarr`",
		`echo 'Error from server (NotFound): secrets not found' >&2; exit 1`: "Secret `bbe-media-api-keys` has no key `sonarr`",
		`echo 'Error from server (Forbidden)' >&2; exit 1`:                   "Failed to read secret `bbe-media-api-keys`: exit status 1\nError from server (Forbidden)",
		"echo 'not json'": "Failed to read secret `bbe-media-api-keys`: invalid character",
		`printf '%s' '{"data": {"sonarr": "%%%"}}'`: "Failed to read key `sonarr` of secret `bbe-media-api-keys`",
	}

	for script, expected := range cases {
		fakeKubectl(t, map[string]string{"get secret": script})

		_, err := KubernetesService{}.GetSecretValue("context", "bbe-media", "bbe-media-api-keys", "sonarr")

		assert.ErrorContains(t, err, expected, script)
	}
}

func Test_SetSecretValues_Succeeds_ReplacingAnExistingSecret(t *testing.T) {
	calls := fakeKubectl(t, map[string]string{
		"get secret": "printf '%s' '" + secretJson + "'",
		"replace":    "true",
	})

	err := KubernetesService{}.SetSecretValues("context", "bbe-media", "bbe-media-indexers", map[string]string{"beyondhd.apiKey": "new"})

	assert.NoError(t, err)
	assert.Equal(t, "replace --filename - --context context", (*calls)[1].args)
	var manifest map[string]interface{}
	assert.NoError(t, json.Unmarshal([]byte((*calls)[1].stdin()), &manifest))
	assert.Equal(t, map[string]interface{}{"prowlarr": "a2V5", "beyondhd.apiKey": "bmV3"}, manifest["data"])
	assert.Equal(t, map[string]interface{}{
		"name":      "bbe-media-indexers",
		"namespace": "bbe-media",
		"labels":    map[string]interface{}{"app.kubernetes.io/managed-by": "bbe"},
	}, manifest["metadata"])
}

func Test_SetSecretValues_Succeeds_CreatingTheSecret(t *testing.T) {
	calls := fakeKubectl(t, map[string]string{
		"get secret": `echo 'Error from server (NotFound): secrets not found' >&2; exit 1`,
		"create":     "true",
	})

	err := KubernetesService{}.SetSecretValues("context", "bbe-media", "bbe-media-indexers", map[string]string{"beyondhd.apiKey": "new"})

	assert.NoError(t, err)
	assert.Equal(t, "create --filename - --context context", (*calls)[1].args)
	assert.Contains(t, (*calls)[1].stdin(), `"data":{"beyondhd.apiKey":"bmV3"}`)
}

func Test_SetSecretValues_Fails(t *testing.T) {
	fakeKubectl(t, map[string]string{"get secret": `echo 'Error from server (Forbidden)' >&2; exit 1`})
	err := KubernetesService{}.SetSecretValues("context", "bbe-media", "bbe-media-indexers", map[string]string{"a": "b"})
	assert.ErrorContains(t, err, "Failed to read secret `bbe-media-indexers`")

	fakeKubectl(t, map[string]string{"get secret": "printf '%s' '" + secretJson + "'", "replace": `echo 'Error from server (Invalid)' >&2; exit 1`})
	err = KubernetesService{}.SetSecretValues("context", "bbe-media", "bbe-media-indexers", map[string]string{"a": "b"})
	assert.EqualError(t, err, "Failed to save secret `bbe-media-indexers`: exit status 1\nError from server (Invalid)")
}

func Test_RemoveSecretValues_Succeeds(t *testing.T) {
	calls := fakeKubectl(t, map[string]string{
		"get secret": "printf '%s' '" + secretJson + "'",
		"replace":    "true",
	})

	err := KubernetesService{}.RemoveSecretValues("context", "bbe-media", "bbe-media-indexers", []string{"beyondhd.apiKey"})

	assert.NoError(t, err)
	assert.Contains(t, (*calls)[1].stdin(), `"data":{"prowlarr":"a2V5"}`)
}

func Test_RemoveSecretValues_Succeeds_WithoutTheSecret(t *testing.T) {
	calls := fakeKubectl(t, map[string]string{"get secret": `echo 'Error from server (NotFound): secrets not found' >&2; exit 1`})

	err := KubernetesService{}.RemoveSecretValues("context", "bbe-media", "bbe-media-indexers", []string{"beyondhd.apiKey"})

	assert.NoError(t, err)
	assert.Len(t, *calls, 1)
}

func Test_NamespaceVolumes_Succeeds(t *testing.T) {
	calls := fakeKubectl(t, map[string]string{
		"get namespace": "echo namespace/bbe-media",
		"get persistentvolumeclaims": `printf '%s' '{"items": [{"metadata": {"name": "bbe-media-sonarr-config"}, "spec": {"resources": {"requests": {"storage": "1Gi"}}}}]}'`,
	})

	volumes, exists, err := KubernetesService{}.NamespaceVolumes("context", "bbe-media")

	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, []models.NamespaceVolume{{Name: "bbe-media-sonarr-config", Size: "1Gi"}}, volumes)
	assert.Equal(t, "get persistentvolumeclaims --namespace bbe-media --output json --context context", (*calls)[1].args)
}

func Test_NamespaceVolumes_Succeeds_WithoutTheNamespace(t *testing.T) {
	fakeKubectl(t, map[string]string{"get namespace": `echo 'Error from server (NotFound): namespaces "bbe-media" not found' >&2; exit 1`})

	volumes, exists, err := KubernetesService{}.NamespaceVolumes("context", "bbe-media")

	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, volumes)
}

func Test_NamespaceVolumes_Fails(t *testing.T) {
	cases := map[string]map[string]string{
		"Failed to read namespace `bbe-media`": {"get namespace": `echo 'Error from server (Forbidden)' >&2; exit 1`},
		"Failed to read the volumes in namespace `bbe-media`: exit status 1": {
			"get namespace":              "echo namespace/bbe-media",
			"get persistentvolumeclaims": `echo 'Error from server (Forbidden)' >&2; exit 1`,
		},
		"Failed to read the volumes in namespace `bbe-media`: invalid character": {
			"get namespace":              "echo namespace/bbe-media",
			"get persistentvolumeclaims": "echo 'not json'",
		},
	}

	for expected, answers := range cases {
		fakeKubectl(t, answers)

		_, _, err := KubernetesService{}.NamespaceVolumes("context", "bbe-media")

		assert.ErrorContains(t, err, expected, expected)
	}
}

func Test_Patch_Succeeds(t *testing.T) {
	calls := fakeKubectl(t, map[string]string{"patch": "true"})

	err := KubernetesService{}.Patch("context", "longhorn-system", "settings.longhorn.io", "deleting-confirmation-flag", `{"value": "true"}`)

	assert.NoError(t, err)
	assert.Equal(t, `patch settings.longhorn.io deleting-confirmation-flag --namespace longhorn-system --type merge --patch {"value": "true"} --context context`, (*calls)[0].args)
}

func Test_Patch_Fails(t *testing.T) {
	fakeKubectl(t, map[string]string{"patch": `echo 'Error from server (NotFound)' >&2; exit 1`})

	err := KubernetesService{}.Patch("context", "longhorn-system", "settings.longhorn.io", "deleting-confirmation-flag", `{}`)

	assert.EqualError(t, err, "Failed to patch settings.longhorn.io `deleting-confirmation-flag`: exit status 1\nError from server (NotFound)")
}

func Test_DeleteNamespace_Succeeds(t *testing.T) {
	calls := fakeKubectl(t, map[string]string{"delete namespace": "true"})

	err := KubernetesService{}.DeleteNamespace("context", "bbe-media")

	assert.NoError(t, err)
	assert.Equal(t, "delete namespace bbe-media --ignore-not-found --wait --timeout 10m --context context", (*calls)[0].args)
}

func Test_DeleteNamespace_Fails(t *testing.T) {
	fakeKubectl(t, map[string]string{"delete namespace": `echo 'error: timed out waiting for the condition' >&2; exit 1`})

	err := KubernetesService{}.DeleteNamespace("context", "bbe-media")

	assert.EqualError(t, err, "Failed to delete namespace `bbe-media`: exit status 1\nerror: timed out waiting for the condition")
}
