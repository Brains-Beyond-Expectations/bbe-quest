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
// The Chart.yaml annotation setting the pod security level of the namespace a chart is installed in
var PodSecurityAnnotation = "bbe/pod-security"
const DefaultBbeLibraryUrl = "https://raw.githubusercontent.com/Brains-Beyond-Expectations/bbe-charts/main/library.yaml"

// Where the package library is read from, a URL or a file. Set with --library or BBE_LIBRARY to test charts that
// aren't released yet
var BbeLibraryUrl = DefaultBbeLibraryUrl
