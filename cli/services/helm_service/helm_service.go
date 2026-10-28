package helm_service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

var execCommand = exec.Command
var osMkdirTemp = os.MkdirTemp
var yamlMarshal = yaml.Marshal

type HelmService struct{}

func (HelmService HelmService) AddRepo(repoName string, repoUrl string) error {
	cmd := execCommand("helm", "repo", "add", repoName, repoUrl)
	logger.Debug(fmt.Sprintf("Adding helm repository `%s` with url `%s`", repoName, repoUrl))
	response, err := cmd.CombinedOutput()
	logger.Debug(fmt.Sprintf("Response: %s", string(response)))

	if err != nil {
		return fmt.Errorf("Failed to add helm repository `%s`: %w", repoName, err)
	}

	updateRepoErr := HelmService.updateRepo(repoName)
	if updateRepoErr != nil {
		return fmt.Errorf("Failed to update helm repository `%s`: %w", repoName, updateRepoErr)
	}

	return nil
}

func (HelmService HelmService) InstallChart(pkgName string, chartName string, repoName string, version string, namespace string, context string, values map[string]interface{}) error {
	// --replace installs over a release whose last install failed, which helm otherwise keeps the name for
	cmd, err := helmCommandWithValues(values, append(append([]string{"install", pkgName}, chartArgs(chartName, repoName, version)...),
		"--namespace", namespace,
		"--create-namespace",
		"--replace",
		"--kube-context", context)...)
	if err != nil {
		return fmt.Errorf("Failed to install helm package `%s`: %w", pkgName, err)
	}
	logger.Debug(fmt.Sprintf("Installing helm chart `%s` from repo `%s` with version `%s` in namespace `%s`", pkgName, repoName, version, namespace))
	logger.Debug(fmt.Sprintf("Command: %s", cmd.String()))

	response, err := cmd.CombinedOutput()
	logger.Debug(fmt.Sprintf("Response: %s", string(response)))

	if err != nil {
		return fmt.Errorf("Failed to install helm package `%s`: %w%s", pkgName, err, helmOutput(response))
	}
	showNotes(pkgName, response)
	return nil
}

func (HelmService HelmService) UpgradeChart(pkgName string, chartName string, repoName string, version string, namespace string, context string, values map[string]interface{}) error {
	cmd, err := helmCommandWithValues(values, append(append([]string{"upgrade", pkgName}, chartArgs(chartName, repoName, version)...),
		"--namespace", namespace,
		"--create-namespace",
		"--kube-context", context)...)
	if err != nil {
		return fmt.Errorf("Failed to upgrade helm package `%s`: %w", pkgName, err)
	}

	response, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to upgrade helm package `%s`: %w%s", pkgName, err, helmOutput(response))
	}
	showNotes(pkgName, response)
	return nil
}

// A chart in a repository is installed by name and version, a chart folder (no repository) as it is in the folder
func chartArgs(chartName string, repoName string, version string) []string {
	if repoName == "" {
		return []string{chartName, "--dependency-update"}
	}

	return []string{fmt.Sprintf("%s/%s", repoName, chartName), "--version", version}
}

// Charts use template functions older helm versions can't parse, such as `break` in Traefik's chart
const minimumHelmVersion = "v3.10.0"

// Every package command pulls its chart first, so helm's version is checked there, once
var helmVersionChecked = false

// Refuses a helm that says it's too old, as it would fail on the chart with an unclear error. A version that can't be
// read doesn't stop anything, helm still explains what it can't do
func verifyHelmVersion() error {
	if helmVersionChecked {
		return nil
	}
	helmVersionChecked = true

	response, err := execCommand("helm", "version", "--template", "{{.Version}}").Output()
	if err != nil {
		logger.Debug(fmt.Sprintf("Could not read helm's version: %v", err))
		return nil
	}

	version := strings.TrimSpace(string(response))
	if semver.IsValid(version) && semver.Compare(version, minimumHelmVersion) < 0 {
		return fmt.Errorf("helm %s is too old for bbe's packages, please upgrade it to %s or newer", version, minimumHelmVersion)
	}

	return nil
}

// Downloads a chart without adding its repository, or packages a chart folder (a file:// URL), to read its values
// and schema
func (HelmService HelmService) PullChart(repositoryUrl string, chartName string, version string) (*models.HelmChart, error) {
	if err := verifyHelmVersion(); err != nil {
		return nil, err
	}

	destination, err := osMkdirTemp("", "bbe-chart-")
	if err != nil {
		return nil, fmt.Errorf("Failed to create a directory for helm chart `%s`: %w", chartName, err)
	}
	defer os.RemoveAll(destination)

	cmd := execCommand("helm", "pull", chartName,
		"--repo", repositoryUrl,
		"--version", version,
		"--destination", destination)
	if path, local := strings.CutPrefix(repositoryUrl, "file://"); local {
		cmd = execCommand("helm", "package", path,
			"--dependency-update",
			"--destination", destination)
	}
	logger.Debug(fmt.Sprintf("Pulling helm chart `%s` with version `%s` from `%s`", chartName, version, repositoryUrl))

	response, err := cmd.CombinedOutput()
	logger.Debug(fmt.Sprintf("Response: %s", string(response)))

	if err != nil {
		return nil, fmt.Errorf("Failed to pull helm chart `%s`: %w%s", chartName, err, helmOutput(response))
	}

	archives, _ := filepath.Glob(filepath.Join(destination, "*.tgz"))
	if len(archives) != 1 {
		return nil, fmt.Errorf("Failed to find the archive for helm chart `%s`", chartName)
	}

	chart, err := readChartArchive(archives[0])
	if err != nil {
		return nil, fmt.Errorf("Failed to read helm chart `%s`: %w", chartName, err)
	}

	return chart, nil
}

func (HelmService HelmService) UninstallChart(pkgName string, namespace string, context string) error {
	// Waiting lets the chart's own clean-up, such as Longhorn's, finish before its namespace goes
	cmd := execCommand("helm", "uninstall", pkgName,
		"--namespace", namespace,
		"--wait",
		"--timeout", "10m",
		"--kube-context", context)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to uninstall helm package `%s`: %w", pkgName, err)
	}
	return nil
}

// A release whose first install failed doesn't count, as nothing in it can be relied on. One whose upgrade failed still runs the version before
func (HelmService HelmService) IsPackageInstalled(pkgName string, namespace string, context string) bool {
	cmd := execCommand("helm", "status", pkgName,
		"--namespace", namespace,
		"--kube-context", context,
		"--output", "json",
	)
	response, err := cmd.Output()
	if err != nil {
		return false
	}

	var release struct {
		Version int `json:"version"`
		Info    struct {
			Status string `json:"status"`
		} `json:"info"`
	}
	if err := json.Unmarshal(response, &release); err != nil {
		return true
	}

	if release.Version == 1 && release.Info.Status == "failed" {
		logger.Debug(fmt.Sprintf("Package `%s` failed to install before, so it counts as not installed", pkgName))
		return false
	}

	return true
}

// The values a package was installed or last upgraded with, as bbe gave them to helm, without the chart's defaults
func (HelmService HelmService) GetValues(pkgName string, namespace string, context string) (map[string]interface{}, error) {
	cmd := execCommand("helm", "get", "values", pkgName,
		"--namespace", namespace,
		"--kube-context", context,
		"--output", "json",
	)
	response, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("Failed to read the values of helm package `%s`: %w", pkgName, err)
	}

	// Helm answers `null` for a package installed without values
	var values map[string]interface{}
	if err := json.Unmarshal(response, &values); err != nil {
		return nil, fmt.Errorf("Failed to read the values of helm package `%s`: %w", pkgName, err)
	}

	return values, nil
}

// The labels the pod security admission reads, see https://kubernetes.io/docs/concepts/security/pod-security-admission
var podSecurityModes = []string{"enforce", "audit", "warn"}

// Creates the namespace with the pod security level a chart needs, as helm's --create-namespace can't label it
func (HelmService HelmService) PrepareNamespace(namespace string, context string, podSecurity string) error {
	logger.Debug(fmt.Sprintf("Setting the pod security level of namespace `%s` to `%s`", namespace, podSecurity))

	response, err := kubectl(context, "create", "namespace", namespace)
	if err != nil && !strings.Contains(string(response), "AlreadyExists") {
		return fmt.Errorf("Failed to create namespace `%s`: %w", namespace, err)
	}

	args := []string{"label", "namespace", namespace, "--overwrite"}
	for _, mode := range podSecurityModes {
		args = append(args, fmt.Sprintf("pod-security.kubernetes.io/%s=%s", mode, podSecurity))
	}
	if _, err := kubectl(context, args...); err != nil {
		return fmt.Errorf("Failed to set the pod security level of namespace `%s`: %w", namespace, err)
	}

	return nil
}

// Helm installs the CRDs in a chart's crds/ folder only on the first install, and never changes them after. Applying
// them first lets an upgrade bring new or changed CRDs, and a reinstall replace the ones an older version left behind.
// Server-side apply takes over the fields helm set, and has no size limit for large CRDs
func (HelmService HelmService) ApplyCrds(context string, crds []string) error {
	cmd := execCommand("kubectl", "apply", "--server-side", "--force-conflicts", "--field-manager", "bbe", "-f", "-", "--context", context)
	cmd.Stdin = strings.NewReader(strings.Join(crds, "\n---\n"))
	logger.Debug(fmt.Sprintf("Applying %d CRD manifests", len(crds)))
	logger.Debug(fmt.Sprintf("Command: %s", cmd.String()))

	response, err := cmd.CombinedOutput()
	logger.Debug(fmt.Sprintf("Response: %s", string(response)))
	if errors.Is(err, exec.ErrNotFound) {
		return errors.New("kubectl is needed to apply the CRDs of this package, please install it")
	}
	if err != nil {
		return fmt.Errorf("Failed to apply the CRDs of this package: %w%s", err, helmOutput(response))
	}

	return nil
}

func kubectl(context string, args ...string) ([]byte, error) {
	cmd := execCommand("kubectl", append(args, "--context", context)...)
	logger.Debug(fmt.Sprintf("Command: %s", cmd.String()))

	response, err := cmd.CombinedOutput()
	logger.Debug(fmt.Sprintf("Response: %s", string(response)))
	if errors.Is(err, exec.ErrNotFound) {
		return response, errors.New("kubectl is needed to prepare the namespace for this package, please install it")
	}
	if err != nil {
		return response, fmt.Errorf("%w%s", err, helmOutput(response))
	}

	return response, nil
}

func (HelmService HelmService) updateRepo(repoName string) error {
	cmd := execCommand("helm", "repo", "update", repoName)
	logger.Debug(fmt.Sprintf("Updating helm repository `%s`", repoName))
	response, err := cmd.CombinedOutput()
	logger.Debug(fmt.Sprintf("Response: %s", string(response)))

	if err != nil {
		return fmt.Errorf("Failed to update helm repository `%s`: %w", repoName, err)
	}

	return nil
}

// Values are passed on stdin, so no temporary file is needed
func helmCommandWithValues(values map[string]interface{}, args ...string) (*exec.Cmd, error) {
	if len(values) == 0 {
		return execCommand("helm", args...), nil
	}

	content, err := yamlMarshal(values)
	if err != nil {
		return nil, err
	}

	cmd := execCommand("helm", append(args, "--values", "-")...)
	cmd.Stdin = bytes.NewReader(content)

	return cmd, nil
}

// Helm explains why a command failed in its output, which is more useful than the exit status
func helmOutput(response []byte) string {
	output := strings.TrimSpace(string(response))
	if output == "" {
		return ""
	}

	return "\n" + output
}

// Charts explain what to do next in their notes, such as how to read a password they generated
func showNotes(pkgName string, response []byte) {
	if notes := chartNotes(response); notes != "" {
		logger.Info(fmt.Sprintf("Notes from `%s`:\n%s", pkgName, notes))
	}
}

func chartNotes(response []byte) string {
	_, notes, found := strings.Cut(string(response), "\nNOTES:\n")
	if !found {
		return ""
	}

	return strings.TrimRight(notes, "\n ")
}
