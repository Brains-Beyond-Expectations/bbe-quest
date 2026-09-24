package constants

import "errors"

var Version = "development"
var ConfigExistsError = errors.New("Config already exists")

var ControlplaneConfigFile = "controlplane.yaml"
var WorkerConfigFile = "worker.yaml"
var TalosConfigFile = "talosconfig"
var BbeConfigFile = "bbe.yaml"
// The Chart.yaml annotation listing what a chart needs in the cluster before it can be installed
var PrerequisitesAnnotation = "bbe/prerequisites"
// The Chart.yaml annotation setting the pod security level of the namespace a chart is installed in
var PodSecurityAnnotation = "bbe/pod-security"
var BbeLibraryUrl = "https://raw.githubusercontent.com/Brains-Beyond-Expectations/bbe-charts/main/library.yaml"
