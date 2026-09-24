package interfaces

import "github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"

type TalosServiceInterface interface {
	BootstrapCluster(helperService HelperServiceInterface, nodeIp string, controlPlaneIp string) error
	DownloadKubeConfig(helperService HelperServiceInterface, nodeIp string, controlPlaneIp string) error
	GenerateConfig(helperService HelperServiceInterface, controlPlaneIp string, clusterName string, talosVersion string) error
	GetControlPlaneIp(helperService HelperServiceInterface, configFile string) (string, error)
	GetDisks(nodeIp string) ([]models.TalosDisk, error)
	GetLocalVersion() (string, error)
	GetNetworkInterface(helperService HelperServiceInterface, nodeIp string) (string, error)
	JoinCluster(helperService HelperServiceInterface, nodeIp string, nodeConfigFile string) error
	ModifyConfigDisk(helperService HelperServiceInterface, configFile string, disk string) error
	ModifyNetworkGateway(helperService HelperServiceInterface, configFile string, gatewayIp string) error
	ModifyNetworkHostname(helperService HelperServiceInterface, configFile string, hostname string) error
	ModifyNetworkInterface(helperService HelperServiceInterface, configFile string, networkInterfaceName string) error
	ModifyDnsServer(helperService HelperServiceInterface, configFile string, dnsServer string) error
	ModifyNetworkNodeIp(helperService HelperServiceInterface, configFile string, nodeIp string) error
	ModifySchedulingOnControlPlane(helperService HelperServiceInterface, allowScheduling bool) error
	ModifyTimeServer(helperService HelperServiceInterface, configFile string, timeServer string) error
	ModifyUserVolumes(helperService HelperServiceInterface, configFile string, names []string) error
	Ping(nodeIp string) bool
	VerifyNodeHealth(helperService HelperServiceInterface, nodeIp string, controlPlaneIp string) error
	WarnIfTalosVersionMismatch(expectedVersion string)
}
