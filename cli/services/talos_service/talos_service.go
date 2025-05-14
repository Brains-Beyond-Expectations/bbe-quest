package talos_service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/go-viper/mapstructure/v2"
	"gopkg.in/yaml.v2"
)

var execCommand = exec.Command
var tenSeconds = 10 * time.Second
var fiveMinutes = 5 * time.Minute

var osReadFile = os.ReadFile
var osWriteFile = os.WriteFile

type TalosService struct{}

func (talosService TalosService) Ping(nodeIp string) bool {
	// First check that this is a Talos device by querying for disks
	cmd := execCommand("talosctl", "-n", nodeIp, "get", "disks", "--insecure")
	output, err := cmd.CombinedOutput()
	logger.Debug(string(output))

	if err != nil {
		return false
	}

	// If it is check if we get turned away by the machineconfig (if so it is likely to be in maintenance mode)
	cmd = execCommand("talosctl", "-n", nodeIp, "get", "machineconfig", "--insecure")
	output, err = cmd.CombinedOutput()
	logger.Debug(string(output))

	return err != nil
}

func (talosService TalosService) GenerateConfig(helperService interfaces.HelperServiceInterface, controlPlaneIp string, clusterName string) error {
	cmd := execCommand("talosctl", "gen", "config", clusterName, fmt.Sprintf("https://%s:6443", controlPlaneIp), "--output", helperService.GetConfigDir())
	output, err := cmd.CombinedOutput()
	logger.Debug(string(output))

	if err != nil {
		if strings.Contains(string(output), "already exists") {
			return constants.ConfigExistsError
		}
		return err
	}

	return nil
}

func (talosService TalosService) JoinCluster(helperService interfaces.HelperServiceInterface, nodeIp string, nodeConfigFile string) error {
	logger.Infof("Instance %s is joining the cluster", nodeIp)

	cmd := execCommand("talosctl", "apply-config", "--insecure", "-n", nodeIp, "--file", helperService.GetConfigFilePath(nodeConfigFile))
	output, err := cmd.CombinedOutput()
	logger.Debug(string(output))

	if err != nil {
		return err
	}

	return nil
}

func (talosService TalosService) BootstrapCluster(helperService interfaces.HelperServiceInterface, nodeIp string, controlPlaneIp string) error {
	logger.Info("Bootstrapping cluster, this might take a few minutes...")

	configFilePath := helperService.GetConfigFilePath(constants.TalosConfigFile)

	start := time.Now()
	timeout := fiveMinutes
	for {
		cmd := execCommand("talosctl", "bootstrap", "--nodes", nodeIp, "--endpoints", controlPlaneIp, fmt.Sprintf("--talosconfig=%s", configFilePath))
		output, err := cmd.CombinedOutput()
		logger.Debug(string(output))

		if err == nil {
			return nil
		}

		if time.Since(start) > timeout {
			return fmt.Errorf("Bootstrap failed after 5 minutes: %w", err)
		}

		time.Sleep(tenSeconds)
	}
}

func (talosService TalosService) VerifyNodeHealth(helperService interfaces.HelperServiceInterface, nodeIp string, controlPlaneIp string) error {
	logger.Info("Verifying cluster health, this might take a few minutes...")

	configFilePath := helperService.GetConfigFilePath(constants.TalosConfigFile)

	start := time.Now()
	timeout := fiveMinutes
	for {
		cmd := execCommand("talosctl", "--nodes", nodeIp, "--endpoints", controlPlaneIp, "health", fmt.Sprintf("--talosconfig=%s", configFilePath))
		output, err := cmd.CombinedOutput()
		logger.Debug(string(output))

		if err == nil {
			return nil
		}

		if time.Since(start) > timeout {
			return fmt.Errorf("Cluster health check failed after 5 minutes: %w", err)
		}

		time.Sleep(tenSeconds)
	}
}

func (talosService TalosService) GetDisks(helperService interfaces.HelperServiceInterface, nodeIp string) ([]string, error) {
	cmd := execCommand("bash", "-c", fmt.Sprintf(`talosctl -n %s get disks --insecure`, nodeIp))
	output, err := cmd.CombinedOutput()
	logger.Debug(string(output))

	if err != nil {
		return nil, err
	}

	disks := strings.Split(string(output), "\n")
	disks = helperService.DeleteEmptyStrings(disks)

	return disks, nil
}

func (talosService TalosService) GetNetworkInterface(helperService interfaces.HelperServiceInterface, nodeIp string) (string, error) {
	cmd := execCommand("bash", "-c", fmt.Sprintf(`talosctl -n %s get addresses --talosconfig %s  --insecure |  awk '$0 ~ /%s/ {print $0}' | awk '{$1=""; print $NF}' | awk '{print $NF}'`, nodeIp, helperService.GetConfigFilePath(constants.TalosConfigFile), nodeIp))
	output, err := cmd.CombinedOutput()
	logger.Debug(string(output))

	if err != nil {
		return "", err
	}

	parsedOutput := strings.TrimSpace(string(output))
	return parsedOutput, nil
}

func (talosService TalosService) ModifyNetworkInterface(helperService interfaces.HelperServiceInterface, configFile string, networkInterfaceName string) error {
	configDir := helperService.GetConfigDir()

	parsedConfig, err := getParsedConfig(configDir, configFile)
	if err != nil {
		return err
	}

	if len(parsedConfig.Machine.Network.Interfaces) == 0 {
		parsedConfig.Machine.Network.Interfaces = append(parsedConfig.Machine.Network.Interfaces, models.TalosInterface{})
	}

	parsedConfig.Machine.Network.Interfaces[0].Interface = networkInterfaceName

	return writeConfig(configDir, configFile, *parsedConfig)
}

func (talosService TalosService) ModifyNetworkGateway(helperService interfaces.HelperServiceInterface, configFile string, gatewayIp string) error {
	configDir := helperService.GetConfigDir()

	parsedConfig, err := getParsedConfig(configDir, configFile)
	if err != nil {
		return err
	}

	if len(parsedConfig.Machine.Network.Interfaces) == 0 {
		parsedConfig.Machine.Network.Interfaces = append(parsedConfig.Machine.Network.Interfaces, models.TalosInterface{})
	}

	routes := []models.TalosRoute{
		{
			Network: "0.0.0.0/0",
			Gateway: gatewayIp,
		},
	}

	parsedConfig.Machine.Network.Interfaces[0].Routes = routes

	return writeConfig(configDir, configFile, *parsedConfig)
}

func (talosService TalosService) ModifyNetworkNodeIp(helperService interfaces.HelperServiceInterface, configFile string, nodeIp string) error {
	configDir := helperService.GetConfigDir()

	parsedConfig, err := getParsedConfig(configDir, configFile)
	if err != nil {
		return err
	}

	if len(parsedConfig.Machine.Network.Interfaces) == 0 {
		parsedConfig.Machine.Network.Interfaces = append(parsedConfig.Machine.Network.Interfaces, models.TalosInterface{})
	}

	addresses := []string{nodeIp}

	parsedConfig.Machine.Network.Interfaces[0].Addresses = addresses

	return writeConfig(configDir, configFile, *parsedConfig)
}

func (talosService TalosService) ModifyNetworkHostname(helperService interfaces.HelperServiceInterface, configFile string, hostname string) error {
	configDir := helperService.GetConfigDir()

	parsedConfig, err := getParsedConfig(configDir, configFile)
	if err != nil {
		return err
	}

	parsedConfig.Machine.Network.Hostname = hostname

	return writeConfig(configDir, configFile, *parsedConfig)
}

func (talosService TalosService) ModifyConfigDisk(helperService interfaces.HelperServiceInterface, configFile string, disk string) error {
	configDir := helperService.GetConfigDir()

	parsedConfig, err := getParsedConfig(configDir, configFile)
	if err != nil {
		return err
	}

	parsedConfig.Machine.Install.Disk = disk

	return writeConfig(configDir, configFile, *parsedConfig)
}

func (talosService TalosService) ModifySchedulingOnControlPlane(helperService interfaces.HelperServiceInterface, allowScheduling bool) error {
	configDir := helperService.GetConfigDir()
	configFile := constants.ControlplaneConfigFile

	parsedConfig, err := getParsedConfig(configDir, configFile)
	if err != nil {
		return err
	}

	parsedConfig.Cluster.AllowSchedulingOnControlPlanes = allowScheduling

	return writeConfig(configDir, configFile, *parsedConfig)
}

func (talosService TalosService) GetControlPlaneIp(helperService interfaces.HelperServiceInterface, configFile string) (string, error) {
	configDir := helperService.GetConfigDir()

	config, err := getParsedConfig(configDir, configFile)
	if err != nil {
		return "", err
	}

	endpoint := config.Cluster.ControlPlane.Endpoint
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimSuffix(endpoint, ":6443")

	return endpoint, nil
}

func (talosService TalosService) DownloadKubeConfig(helperService interfaces.HelperServiceInterface, nodeIp string, controlPlaneIp string) error {
	cmd := execCommand("talosctl", "kubeconfig", "--nodes", nodeIp, "--endpoints", controlPlaneIp, fmt.Sprintf("--talosconfig=%s", helperService.GetConfigFilePath(constants.TalosConfigFile)))
	output, err := cmd.CombinedOutput()
	logger.Debug(string(output))

	if err != nil {
		return err
	}

	return nil
}

func getParsedConfig(configDir string, configFile string) (*models.TalosMachineConfig, error) {
	initialTalosConfig, err := osReadFile(fmt.Sprintf("%s/%s", configDir, configFile))
	if err != nil {
		panic(err)
	}

	var yamlConfig map[string]interface{}
	err = yaml.Unmarshal(initialTalosConfig, &yamlConfig)
	if err != nil {
		return nil, err
	}

	var parsedConfig models.TalosMachineConfig
	err = mapstructure.Decode(yamlConfig, &parsedConfig)
	if err != nil {
		return nil, err
	}

	return &parsedConfig, nil
}

func writeConfig(configDir string, configFile string, parsedConfig models.TalosMachineConfig) error {
	mappedConfig := make(map[string]interface{})
	err := mapstructure.Decode(&parsedConfig, &mappedConfig)
	if err != nil {
		return err
	}

	configToWrite, err := yaml.Marshal(mappedConfig)
	if err != nil {
		return err
	}

	return osWriteFile(fmt.Sprintf("%s/%s", configDir, configFile), configToWrite, 0644)
}
