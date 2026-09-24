package prerequisite_service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
)

var cluster = models.ClusterAccess{KubeContext: "admin@test", ControlPlaneIp: "192.168.1.161", TalosConfig: "/config/talosconfig"}

const nodesJson = `{"items": [
  {"metadata": {"name": "control-1", "labels": {"node-role.kubernetes.io/control-plane": ""}},
   "status": {"addresses": [{"type": "Hostname", "address": "control-1"}, {"type": "InternalIP", "address": "192.168.1.161"}]}},
  {"metadata": {"name": "worker-1", "labels": {}},
   "status": {"addresses": [{"type": "InternalIP", "address": "192.168.1.162"}]}}
]}`

// What `talosctl get extensions -o yaml` prints: one document per extension
const extensionsYaml = `node: 192.168.1.161
metadata:
    namespace: runtime
    type: ExtensionStatuses.runtime.talos.dev
    id: "0"
spec:
    image: 000.iscsi-tools.sqsh
    metadata:
        name: iscsi-tools
        version: v0.2.0
---
node: 192.168.1.161
metadata:
    id: "1"
spec:
    image: 001.schematic.sqsh
    metadata:
        name: schematic
        version: b879cae68366c21e706bb4f1ee0f15fda6d8be38375ae144cb9978b9df0e3caf
`

// What `talosctl get machineconfig -o yaml` prints for a config made for Talos 1.14, with the config as a string
const newConfigYaml = `node: 192.168.1.161
metadata:
    namespace: config
    type: MachineConfigs.config.talos.dev
    id: v1alpha1
spec: |
    version: v1alpha1
    machine:
        type: worker
    ---
    apiVersion: v1alpha1
    kind: KubeletConfig
    image: ghcr.io/siderolabs/kubelet:v1.37.0
    ---
    apiVersion: v1alpha1
    kind: UserVolumeConfig
    name: other
    volumeType: directory
`

// A config made for Talos 1.13 or older, which sets up the kubelet in the v1alpha1 document
const oldConfigYaml = `node: 192.168.1.161
metadata:
    id: v1alpha1
spec: |
    version: v1alpha1
    machine:
        type: worker
        kubelet:
            image: ghcr.io/siderolabs/kubelet:v1.33.0
            extraMounts:
                - destination: /var/mnt/other
                  type: bind
                  source: /var/mnt/other
`

const versionOutput = `Client:
	Tag:         v1.14.0
	SHA:         2f86b9d2
Server:
	NODE:        192.168.1.161
	Tag:         v1.14.1
	SHA:         2f86b9d2
`

type fakeCommand struct {
	output string
	fails  bool
}

// Answers kubectl and talosctl by the start of their arguments, and records every call
func fakeCommands(t *testing.T, answers map[string]fakeCommand) *[]string {
	calls := &[]string{}
	execCommand = func(name string, args ...string) *exec.Cmd {
		call := name + " " + strings.Join(args, " ")
		*calls = append(*calls, call)
		for prefix, answer := range answers {
			if strings.HasPrefix(call, prefix) {
				script := `printf '%s' "$0"`
				if answer.fails {
					script += "; exit 1"
				}
				return exec.Command("sh", "-c", script, answer.output)
			}
		}
		t.Fatalf("unexpected command: %s", call)
		return nil
	}
	t.Cleanup(func() { execCommand = exec.Command })

	return calls
}

func Test_Check_Succeeds_WhenTheStorageClassExists(t *testing.T) {
	calls := fakeCommands(t, map[string]fakeCommand{"kubectl get storageclass longhorn": {output: "storageclass.storage.k8s.io/longhorn"}})

	missing, err := PrerequisiteService{}.Check(models.ChartPrerequisite{StorageClass: "longhorn"}, cluster)

	assert.NoError(t, err)
	assert.Empty(t, missing)
	assert.Equal(t, []string{"kubectl get storageclass longhorn -o name --context admin@test"}, *calls)
}

func Test_Check_Succeeds_ReportingAMissingStorageClass(t *testing.T) {
	fakeCommands(t, map[string]fakeCommand{"kubectl get storageclass": {output: `Error from server (NotFound): storageclasses.storage.k8s.io "longhorn" not found`, fails: true}})

	missing, err := PrerequisiteService{}.Check(models.ChartPrerequisite{StorageClass: "longhorn"}, cluster)

	assert.NoError(t, err)
	assert.Equal(t, []string{"the `longhorn` storage class"}, missing)
}

func Test_Check_Fails_WhenTheClusterCannotBeReached(t *testing.T) {
	fakeCommands(t, map[string]fakeCommand{"kubectl get storageclass": {output: "Unable to connect to the server", fails: true}})

	_, err := PrerequisiteService{}.Check(models.ChartPrerequisite{StorageClass: "longhorn"}, cluster)

	assert.ErrorContains(t, err, "Failed to check for the `longhorn` storage class: exit status 1\nUnable to connect to the server")
}

func Test_Check_Fails_WithoutKubectl(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd { return exec.Command("bbe-test-command-that-does-not-exist") }
	t.Cleanup(func() { execCommand = exec.Command })

	_, err := PrerequisiteService{}.Check(models.ChartPrerequisite{StorageClass: "longhorn"}, cluster)

	assert.ErrorContains(t, err, "kubectl is needed to check this package's prerequisites, please install it")
}

func Test_Check_Succeeds_ReportingWhatEachNodeIsMissing(t *testing.T) {
	calls := fakeCommands(t, map[string]fakeCommand{
		"kubectl get nodes":          {output: nodesJson},
		"talosctl get extensions":    {output: extensionsYaml},
		"talosctl get machineconfig": {output: newConfigYaml},
	})
	prerequisite := models.ChartPrerequisite{Talos: &models.TalosPrerequisite{
		Extensions:  []string{"siderolabs/iscsi-tools", "siderolabs/util-linux-tools"},
		Directories: []string{"/var/mnt/longhorn", "/var/mnt/other"},
	}}

	missing, err := PrerequisiteService{}.Check(prerequisite, cluster)

	assert.NoError(t, err)
	assert.Equal(t, []string{
		"the siderolabs/util-linux-tools extension on node `control-1`",
		"the /var/mnt/longhorn directory on node `control-1`",
		"the siderolabs/util-linux-tools extension on node `worker-1`",
		"the /var/mnt/longhorn directory on node `worker-1`",
	}, missing)
	assert.Contains(t, *calls, "talosctl get extensions -o yaml --nodes 192.168.1.162 --endpoints 192.168.1.161 --talosconfig /config/talosconfig")
}

func Test_PrepareNodes_Succeeds_PatchingAndUpgradingWorkersFirst(t *testing.T) {
	var posted string
	factory := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/schematics/b879cae68366c21e706bb4f1ee0f15fda6d8be38375ae144cb9978b9df0e3caf":
			w.Write([]byte("overlay:\n    image: siderolabs/sbc-raspberrypi\n    name: rpi_generic\ncustomization:\n    systemExtensions:\n        officialExtensions:\n            - siderolabs/iscsi-tools\n"))
		case r.Method == http.MethodPost && r.URL.Path == "/schematics":
			body, _ := io.ReadAll(r.Body)
			posted = string(body)
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id": "f8a903f101ce10f686476024898734bb6b36353cc4d41f348514db9004ec0a9d"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer factory.Close()
	originalUrl := factoryUrl
	factoryUrl = factory.URL
	t.Cleanup(func() { factoryUrl = originalUrl })

	calls := fakeCommands(t, map[string]fakeCommand{
		"kubectl get nodes":          {output: nodesJson},
		"talosctl get extensions":    {output: extensionsYaml},
		"talosctl get machineconfig": {output: newConfigYaml},
		"talosctl patch":             {output: "patched"},
		"talosctl version":           {output: versionOutput},
		"talosctl upgrade":           {output: "upgraded"},
	})

	err := PrerequisiteService{}.PrepareNodes(models.TalosPrerequisite{
		Extensions:  []string{"siderolabs/iscsi-tools", "siderolabs/util-linux-tools"},
		Directories: []string{"/var/mnt/longhorn"},
	}, cluster)

	assert.NoError(t, err)
	var changes []string
	for _, call := range *calls {
		if strings.HasPrefix(call, "talosctl patch") || strings.HasPrefix(call, "talosctl upgrade") {
			changes = append(changes, call)
		}
	}
	image := strings.TrimPrefix(factory.URL, "http://") + "/metal-installer/f8a903f101ce10f686476024898734bb6b36353cc4d41f348514db9004ec0a9d:v1.14.1"
	patch := `{"apiVersion":"v1alpha1","kind":"UserVolumeConfig","name":"longhorn","volumeType":"directory"}`
	assert.Equal(t, []string{
		"talosctl patch machineconfig --patch " + patch + " --nodes 192.168.1.162 --endpoints 192.168.1.161 --talosconfig /config/talosconfig",
		"talosctl upgrade --image " + image + " --wait --timeout 30m --nodes 192.168.1.162 --endpoints 192.168.1.161 --talosconfig /config/talosconfig",
		"talosctl patch machineconfig --patch " + patch + " --nodes 192.168.1.161 --endpoints 192.168.1.161 --talosconfig /config/talosconfig",
		"talosctl upgrade --image " + image + " --wait --timeout 30m --nodes 192.168.1.161 --endpoints 192.168.1.161 --talosconfig /config/talosconfig",
	}, changes)
	// The overlay is kept and only the missing extension is added
	assert.Contains(t, posted, "name: rpi_generic")
	assert.Contains(t, posted, "- siderolabs/iscsi-tools\n")
	assert.Contains(t, posted, "- siderolabs/util-linux-tools\n")
	assert.Equal(t, 1, strings.Count(posted, "iscsi-tools"))
}

func Test_PrepareNodes_Succeeds_WithoutChangesWhenNodesAreReady(t *testing.T) {
	calls := fakeCommands(t, map[string]fakeCommand{
		"kubectl get nodes":          {output: nodesJson},
		"talosctl get extensions":    {output: extensionsYaml},
		"talosctl get machineconfig": {output: newConfigYaml},
	})

	err := PrerequisiteService{}.PrepareNodes(models.TalosPrerequisite{Extensions: []string{"siderolabs/iscsi-tools"}, Directories: []string{"/var/mnt/other"}}, cluster)

	assert.NoError(t, err)
	for _, call := range *calls {
		assert.NotContains(t, call, "patch")
		assert.NotContains(t, call, "upgrade")
	}
}

func Test_PrepareNodes_Fails_WhenANodeIsNotFromImageFactory(t *testing.T) {
	fakeCommands(t, map[string]fakeCommand{
		"kubectl get nodes":       {output: nodesJson},
		"talosctl get extensions": {output: "spec:\n    metadata:\n        name: iscsi-tools\n        version: v0.2.0\n"},
	})

	err := PrerequisiteService{}.PrepareNodes(models.TalosPrerequisite{Extensions: []string{"siderolabs/util-linux-tools"}}, cluster)

	assert.EqualError(t, err, "Node `worker-1` wasn't installed from an Image Factory image, so bbe can't add extensions to it")
}

func Test_PrepareNodes_Fails_WhenImageFactoryFails(t *testing.T) {
	factory := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("factory is down"))
	}))
	defer factory.Close()
	originalUrl := factoryUrl
	factoryUrl = factory.URL
	t.Cleanup(func() { factoryUrl = originalUrl })
	fakeCommands(t, map[string]fakeCommand{
		"kubectl get nodes":       {output: nodesJson},
		"talosctl get extensions": {output: extensionsYaml},
		"talosctl version":        {output: versionOutput},
	})

	err := PrerequisiteService{}.PrepareNodes(models.TalosPrerequisite{Extensions: []string{"siderolabs/util-linux-tools"}}, cluster)

	assert.ErrorContains(t, err, "Failed to build an image with siderolabs/util-linux-tools for node `worker-1`")
	assert.ErrorContains(t, err, "factory is down")
}

func Test_PrepareNodes_Fails_WhenTheUpgradeFails(t *testing.T) {
	factory := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Write([]byte(`{"id": "new"}`))
			return
		}
		w.Write([]byte("customization: {}\n"))
	}))
	defer factory.Close()
	originalUrl := factoryUrl
	factoryUrl = factory.URL
	t.Cleanup(func() { factoryUrl = originalUrl })
	fakeCommands(t, map[string]fakeCommand{
		"kubectl get nodes":       {output: nodesJson},
		"talosctl get extensions": {output: extensionsYaml},
		"talosctl version":        {output: versionOutput},
		"talosctl upgrade":        {output: "error: node didn't come back", fails: true},
	})

	err := PrerequisiteService{}.PrepareNodes(models.TalosPrerequisite{Extensions: []string{"siderolabs/util-linux-tools"}}, cluster)

	assert.ErrorContains(t, err, "Failed to upgrade node `worker-1`")
	assert.ErrorContains(t, err, "node didn't come back")
}

func Test_PrepareNodes_Fails_WhenThePatchFails(t *testing.T) {
	fakeCommands(t, map[string]fakeCommand{
		"kubectl get nodes":          {output: nodesJson},
		"talosctl get machineconfig": {output: newConfigYaml},
		"talosctl patch":             {output: "permission denied", fails: true},
	})

	err := PrerequisiteService{}.PrepareNodes(models.TalosPrerequisite{Directories: []string{"/var/mnt/longhorn"}}, cluster)

	assert.ErrorContains(t, err, "Failed to add /var/mnt/longhorn on node `worker-1`")
}

func Test_listNodes_Fails_WithUnreadableOrIncompleteNodes(t *testing.T) {
	fakeCommands(t, map[string]fakeCommand{"kubectl get nodes": {output: "not json"}})
	_, err := listNodes(cluster)
	assert.ErrorContains(t, err, "Failed to read the cluster's nodes")

	fakeCommands(t, map[string]fakeCommand{"kubectl get nodes": {output: `{"items": [{"metadata": {"name": "node-1"}, "status": {"addresses": []}}]}`}})
	_, err = listNodes(cluster)
	assert.EqualError(t, err, "Node `node-1` has no internal IP address")

	fakeCommands(t, map[string]fakeCommand{"kubectl get nodes": {output: "forbidden", fails: true}})
	_, err = listNodes(cluster)
	assert.ErrorContains(t, err, "Failed to list the cluster's nodes")
}

func Test_parseServerVersion_Succeeds_IgnoringTheClientVersion(t *testing.T) {
	version, err := parseServerVersion(versionOutput)
	assert.NoError(t, err)
	assert.Equal(t, "v1.14.1", version)

	_, err = parseServerVersion("Client:\n\tTag:         v1.14.0\n")
	assert.EqualError(t, err, "Failed to find the node's Talos version")
}

func Test_PrepareNodes_Succeeds_MountingDirectoriesIntoTheKubeletOnOlderConfigs(t *testing.T) {
	calls := fakeCommands(t, map[string]fakeCommand{
		"kubectl get nodes":          {output: nodesJson},
		"talosctl get machineconfig": {output: oldConfigYaml},
		"talosctl patch":             {output: "patched"},
	})
	prerequisite := models.TalosPrerequisite{Directories: []string{"/var/mnt/longhorn", "/var/mnt/other"}}

	missing, err := PrerequisiteService{}.Check(models.ChartPrerequisite{Talos: &prerequisite}, cluster)
	assert.NoError(t, err)
	assert.Equal(t, []string{"the /var/mnt/longhorn directory on node `control-1`", "the /var/mnt/longhorn directory on node `worker-1`"}, missing)

	err = PrerequisiteService{}.PrepareNodes(prerequisite, cluster)

	assert.NoError(t, err)
	patch := `{"machine":{"kubelet":{"extraMounts":[{"destination":"/var/mnt/longhorn","options":["bind","rshared","rw"],"source":"/var/mnt/longhorn","type":"bind"}]}}}`
	assert.Contains(t, *calls, "talosctl patch machineconfig --patch "+patch+" --nodes 192.168.1.162 --endpoints 192.168.1.161 --talosconfig /config/talosconfig")
}

func Test_readNodeConfig_Fails_WithUnreadableConfigs(t *testing.T) {
	testNode := node{name: "worker-1", ip: "192.168.1.162"}

	fakeCommands(t, map[string]fakeCommand{"talosctl get machineconfig": {output: "not running", fails: true}})
	_, err := readNodeConfig(testNode, cluster)
	assert.ErrorContains(t, err, "Failed to read the machine config of node `worker-1`")

	fakeCommands(t, map[string]fakeCommand{"talosctl get machineconfig": {output: "spec: [broken"}})
	_, err = readNodeConfig(testNode, cluster)
	assert.ErrorContains(t, err, "Failed to read the machine config of node `worker-1`")

	fakeCommands(t, map[string]fakeCommand{"talosctl get machineconfig": {output: "spec: |\n    machine: [broken\n"}})
	_, err = readNodeConfig(testNode, cluster)
	assert.ErrorContains(t, err, "Failed to read the machine config of node `worker-1`")
}
