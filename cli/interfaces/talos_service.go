package interfaces

import "github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"

type TalosServiceInterface interface {
	BootstrapCluster(helperService HelperServiceInterface, nodeIp string) error
	DownloadKubeConfig(helperService HelperServiceInterface, talosEndpoints []string) error
	GenerateConfig(helperService HelperServiceInterface, endpointIp string, clusterName string, talosVersion string) error
	GetControlPlaneIp(helperService HelperServiceInterface, configFile string) (string, error)
	GetDisks(nodeIp string) ([]models.TalosDisk, error)
	GetLocalVersion() (string, error)
	GetMembers(helperService HelperServiceInterface, talosEndpoints []string) ([]models.TalosMember, error)
	GetNetworkInterface(helperService HelperServiceInterface, nodeIp string) (string, error)
	GetNodeNetworkInterface(helperService HelperServiceInterface, talosEndpoints []string, nodeIp string) (string, error)
	GetSharedIp(helperService HelperServiceInterface) (string, error)
	GetTalosEndpoints(helperService HelperServiceInterface) ([]string, error)
	JoinCluster(helperService HelperServiceInterface, nodeIp string, nodeConfigFile string) error
	ModifyClusterEndpoint(helperService HelperServiceInterface, configFile string, ip string) error
	ModifySharedIp(helperService HelperServiceInterface, sharedIp string, networkInterface string) error
	MoveNodeToSharedIp(helperService HelperServiceInterface, talosEndpoints []string, nodeIp string, sharedIp string, networkInterface string) error
	ModifyConfigDisk(helperService HelperServiceInterface, configFile string, disk string) error
	ModifyInstallerImage(helperService HelperServiceInterface, configFile string, image string) error
	ModifyFirewall(helperService HelperServiceInterface, configFile string, trustedNetworks []string) error
	ModifyNetworkGateway(helperService HelperServiceInterface, configFile string, gatewayIp string) error
	ModifyNetworkHostname(helperService HelperServiceInterface, configFile string, hostname string) error
	ModifyNetworkInterface(helperService HelperServiceInterface, configFile string, networkInterfaceName string) error
	ModifyControlPlaneMetrics(helperService HelperServiceInterface) error
	ModifyDnsServer(helperService HelperServiceInterface, configFile string, dnsServer string) error
	ModifyNetworkNodeIp(helperService HelperServiceInterface, configFile string, nodeIp string) error
	ModifyNetworkPolicies(helperService HelperServiceInterface) error
	ModifySchedulingOnControlPlane(helperService HelperServiceInterface, allowScheduling bool) error
	ModifyTimeServer(helperService HelperServiceInterface, configFile string, timeServer string) error
	ModifyUserVolumes(helperService HelperServiceInterface, configFile string, names []string) error
	Ping(nodeIp string) bool
	SetTalosEndpoints(helperService HelperServiceInterface, talosEndpoints []string) error
	VerifyNodeHealth(helperService HelperServiceInterface, talosEndpoints []string) error
	WarnIfTalosVersionMismatch(expectedVersion string)
}
