package ipfinder_service

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/Brains-Beyond-Expectations/bbe-quest/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/misc/logger"
)

var execCommand = exec.Command

type IpFinderService struct{}

func (ipFinderService IpFinderService) LocateDevice(helperService interfaces.HelperServiceInterface, talosService interfaces.TalosServiceInterface, ip string) ([]string, error) {
	logger.Infof("Scanning network for Talos devices on %s/24", ip)

	cmd := execCommand("bash", "-c", fmt.Sprintf(`nmap -sn %s/24 -oG - | awk '/Up$/{print $2}'`, ip))
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Debug(string(output))
		return nil, err
	}

	ips := strings.Split(string(output), "\n")

	ips = helperService.DeleteEmptyStrings(ips)

	talosIps := []string{}
	for _, ip := range ips {
		if talosService.Ping(ip) {
			logger.Debug(fmt.Sprintf("Found Talos device at %s", ip))
			talosIps = append(talosIps, ip)
		}
	}

	return talosIps, nil
}

func (ipFinderService IpFinderService) GetGatewayIp(helperService interfaces.HelperServiceInterface) (string, error) {
	var output []byte
	var err error

	command := `netstat -rn | grep -E '^(default|0.0.0.0)' | awk '{print $2}' | head -n 1`
	if helperService.IsWsl() {
		command = `/mnt/c/Windows/System32/ipconfig.exe | grep -E '(192\.168\.|172\.16\.|10\.0\.)' | grep -m1 'Default Gateway' | awk '{print $13}' | tr -d '\r'`
	}

	logger.Info("Trying to determine the Gateway IP address...")
	cmd := execCommand("bash", "-c", command)
	output, err = cmd.CombinedOutput()
	if err != nil {
		logger.Debug(string(output))
		logger.Error("Error while determining IP address", nil)
		return "", nil
	}

	result := strings.TrimSpace(string(output))
	return string(result), nil
}
