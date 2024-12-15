package talos

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/nicolajv/bbe-quest/helper"
	"github.com/sirupsen/logrus"
)

func Ping(nodeIp string) bool {
	// First check that this is a Talos device by querying for disks
	cmd := exec.Command("talosctl", "-n", nodeIp, "disks", "--insecure")
	err := cmd.Run()

	if err != nil {
		return false
	}

	// If it is check if we get turned away by the machineconfig (if so it is likely to be in maintenance mode)
	cmd = exec.Command("talosctl", "-n", nodeIp, "get", "machineconfig", "--insecure")
	err = cmd.Run()

	return err != nil
}

func GenerateConfig(controlPlaneIp string, clusterName string) error {
	cmd := exec.Command("talosctl", "gen", "config", clusterName, fmt.Sprintf("https://%s:6443", controlPlaneIp))
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "already exists") {
			logrus.Info("Config already exists, skipping generation")
			return nil
		}
		return err
	}

	return nil
}

func JoinCluster(nodeIp string, nodeConfigFile string) error {
	logrus.Infof("Instance %s is joining the cluster", nodeIp)

	cmd := exec.Command("talosctl", "apply-config", "--insecure", "-n", nodeIp, "--file", nodeConfigFile)
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func BootstrapCluster(nodeIp string, controlPlaneIp string, talosConfigFile string) error {
	logrus.Info("Bootstrapping cluster")

	start := time.Now()
	timeout := 5 * time.Minute
	for {
		cmd := exec.Command("talosctl", "bootstrap", "--nodes", nodeIp, "--endpoints", controlPlaneIp, fmt.Sprintf("--talosconfig=%s", talosConfigFile))
		err := cmd.Run()
		if err == nil {
			return nil
		}

		if time.Since(start) > timeout {
			return fmt.Errorf("bootstrap failed after 5 minutes: %w", err)
		}

		time.Sleep(10 * time.Second)
	}
}

func VerifyNodeHealth(nodeIp string, controlPlaneIp string, talosConfigFile string) error {
	logrus.Info("Verifying cluster health")

	start := time.Now()
	timeout := 5 * time.Minute
	for {
		cmd := exec.Command("talosctl", "--nodes", nodeIp, "--endpoints", controlPlaneIp, "health", fmt.Sprintf("--talosconfig=%s", talosConfigFile))
		err := cmd.Run()
		if err == nil {
			return nil
		}

		if time.Since(start) > timeout {
			return fmt.Errorf("cluster health check failed after 5 minutes: %w", err)
		}

		time.Sleep(10 * time.Second)
	}
}

func GetDisks(nodeIp string) ([]string, error) {
	cmd := exec.Command("bash", "-c", fmt.Sprintf(`talosctl -n %s disks --insecure | awk 'NR>1 && NF>0 {print $1}'`, nodeIp))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	disks := strings.Split(string(output), "\n")
	disks = helper.DeleteEmptyStrings(disks)

	return disks, nil
}

func ModifyConfigDisk(configFile string, disk string) error {
	cmd := exec.Command("yq", "eval", fmt.Sprintf(`.machine.install.disk = "%s"`, disk), "-i", configFile)
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func GetControlPlaneIp(configFile string) (string, error) {
	cmd := exec.Command("yq", "eval", `.cluster.controlPlane.endpoint | sub("https://", "") | sub(":6443", "")`, configFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
