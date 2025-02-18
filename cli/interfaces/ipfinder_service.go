package interfaces

type IpFinderServiceInterface interface {
	LocateDevice(helperService HelperServiceInterface, talosService TalosServiceInterface, ip string) ([]string, error)
	GetGatewayIp(helperService HelperServiceInterface) (string, error)
}
