package interfaces

type TalosServiceInterface interface {
	Ping(nodeIp string) bool
	GenerateConfig(helperService HelperServiceInterface, controlPlaneIp string, clusterName string) error
	JoinCluster(helperService HelperServiceInterface, nodeIp string, nodeConfigFile string) error
	BootstrapCluster(helperService HelperServiceInterface, nodeIp string, controlPlaneIp string) error
	VerifyNodeHealth(helperService HelperServiceInterface, nodeIp string, controlPlaneIp string) error
	GetDisks(helperService HelperServiceInterface, nodeIp string) ([]string, error)
	GetNetworkInterface(helperService HelperServiceInterface, nodeIp string) (string, error)
	ModifyNetworkInterface(helperService HelperServiceInterface, configFile string, networkInterfaceName string) error
	ModifyNetworkGateway(helperService HelperServiceInterface, configFile string, gatewayIp string) error
	ModifyNetworkNodeIp(helperService HelperServiceInterface, configFile string, nodeIp string) error
	ModifyNetworkHostname(helperService HelperServiceInterface, configFile string, hostname string) error
	ModifyConfigDisk(helperService HelperServiceInterface, configFile string, disk string) error
	ModifySchedulingOnControlPlane(helperService HelperServiceInterface, allowScheduling bool) error
	GetControlPlaneIp(helperService HelperServiceInterface, configFile string) (string, error)
	DownloadKubeConfig(helperService HelperServiceInterface, nodeIp string, controlPlaneIp string) error
}
