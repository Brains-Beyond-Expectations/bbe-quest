package ipfinder

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/nicolajv/bbe-quest/helper"
	"github.com/nicolajv/bbe-quest/services/talos"
	"github.com/nicolajv/bbe-quest/ui"
	"github.com/sirupsen/logrus"
)

func LocateDevice() ([]string, error) {
	ip, err := GetIp()
	if err != nil {
		return nil, err
	}

	logrus.Infof("Scanning network for Talos devices on %s/24", ip)

	cmd := exec.Command("bash", "-c", fmt.Sprintf(`nmap -sn %s/24 -oG - | awk '/Up$/{print $2}'`, ip))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	ips := strings.Split(string(output), "\n")

	ips = helper.DeleteEmptyStrings(ips)

	talosIps := []string{}
	for _, ip := range ips {
		if talos.Ping(ip) {
			logrus.Infof("Found Talos device at %s", ip)
			talosIps = append(talosIps, ip)
		}
	}

	logrus.Infof("Found %d Talos device(s)", len(talosIps))

	return talosIps, nil
}

func GetIp() (string, error) {
	if helper.IsWsl() {
		logrus.Info("WSL detected, trying to determine IP address...")
		cmd := exec.Command("bash", "-c", `/mnt/c/Windows/System32/ipconfig.exe | grep -E '(192\.168\.|172\.16\.|10\.0\.)' | grep -m1 IPv4 | awk '{print $14}' | tr -d '\r'`)
		output, err := cmd.CombinedOutput()
		if err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error("Error while determining IP address")
			return "", err
		}
		result := strings.TrimSpace(string(output))
		if helper.IsValidIp(string(result)) {
			return string(result), nil
		}
	}

	// TODO: Add support for other OS's

	var result string
	title := "IP not found, please enter the IP of the network you want to scan:"
	for {
		var err error
		result, err = ui.CreateInput(title)
		if err != nil {
			return "", err
		}
		if helper.IsValidIp(result) {
			break
		}
		title = "Invalid IP, please enter a valid IP:"
	}
	return result, nil
}
