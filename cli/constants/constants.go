package constants

import "errors"

var Version = "development"
var ConfigExistsError = errors.New("Config already exists")

var ControlplaneConfigFile = "controlplane.yaml"
var WorkerConfigFile = "worker.yaml"
var TalosConfigFile = "talosconfig"
var BbeConfigFile = "bbe.yaml"
// Directory user volumes every node gets under /var/mnt, so storage packages like bbe-storage's Longhorn work on new nodes
var UserVolumes = []string{"longhorn"}
// The Chart.yaml annotation listing what a chart needs in the cluster before it can be installed
var PrerequisitesAnnotation = "bbe/prerequisites"
var BbeLibraryUrl = "https://raw.githubusercontent.com/Brains-Beyond-Expectations/bbe-charts/main/library.yaml"
