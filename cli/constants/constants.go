package constants

import "errors"

var Version = "development"
var ConfigExistsError = errors.New("Config already exists")

var ControlplaneConfigFile = "controlplane.yaml"
var WorkerConfigFile = "worker.yaml"
var TalosConfigFile = "talosconfig"
var BbeConfigFile = "bbe.yaml"
var BbeLibraryUrl = "https://raw.githubusercontent.com/Brains-Beyond-Expectations/bbe-charts/main/library.yaml"
