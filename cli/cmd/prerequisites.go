package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
)

// Makes sure what a package needs in the cluster is there before it's installed or upgraded, by installing
// the package that provides it or setting up the nodes, after asking the user
type prerequisiteRunner struct {
	helperService       interfaces.HelperServiceInterface
	uiService           interfaces.UiServiceInterface
	packageService      interfaces.PackageServiceInterface
	helmService         interfaces.HelmServiceInterface
	talosService        interfaces.TalosServiceInterface
	prerequisiteService interfaces.PrerequisiteServiceInterface
	bbeConfig           models.BbeConfig
	allPackages         []models.ChartEntry
	// Packages the user already picked are installed first when another needs them, without asking again
	chosen      []string
	interactive bool
	// Installs a package that provides a prerequisite, and records it in bbe.yaml
	install func(pkg models.ChartEntry) error
	cluster *models.ClusterAccess
}

func (runner *prerequisiteRunner) ensure(pkg models.ChartEntry) error {
	return runner.ensureFor(pkg, nil)
}

// `requiredBy` lists the packages waiting on this one, to stop at packages that need each other
func (runner *prerequisiteRunner) ensureFor(pkg models.ChartEntry, requiredBy []string) error {
	if slices.Contains(requiredBy, pkg.Name) {
		return fmt.Errorf("Packages need each other: %s", strings.Join(append(requiredBy, pkg.Name), " -> "))
	}

	prerequisites, err := runner.packageService.GetPrerequisites(pkg, runner.helmService)
	if err != nil {
		return err
	}

	for _, prerequisite := range prerequisites {
		if err := runner.ensureOne(pkg, prerequisite, append(slices.Clone(requiredBy), pkg.Name)); err != nil {
			return err
		}
	}

	return nil
}

func (runner *prerequisiteRunner) ensureOne(pkg models.ChartEntry, prerequisite models.ChartPrerequisite, requiredBy []string) error {
	cluster, err := runner.clusterAccess()
	if err != nil {
		return err
	}

	missing, err := runner.prerequisiteService.Check(prerequisite, cluster)
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}

	var provider models.ChartEntry
	if prerequisite.Package != "" {
		index := slices.IndexFunc(runner.allPackages, func(candidate models.ChartEntry) bool { return candidate.Name == prerequisite.Package })
		if index < 0 {
			return fmt.Errorf("Package `%s` needs %s from package `%s`, which isn't in the library", pkg.Name, prerequisite.Name, prerequisite.Package)
		}
		provider = runner.allPackages[index]
	}

	needs := prerequisite.Name
	if prerequisite.Description != "" {
		needs = fmt.Sprintf("%s (%s)", prerequisite.Name, prerequisite.Description)
	}
	logger.Warning(fmt.Sprintf("Package `%s` needs %s, and the cluster is missing %s", pkg.Name, needs, strings.Join(missing, ", ")))

	if !(prerequisite.Package != "" && slices.Contains(runner.chosen, prerequisite.Package)) {
		if !runner.interactive {
			return fmt.Errorf("Package `%s` needs %s. Run without --yes to set it up", pkg.Name, prerequisite.Name)
		}

		answer, err := runner.uiService.CreateSelect(fmt.Sprintf("Set up %s for `%s` now? %s", prerequisite.Name, pkg.Name, describeFix(prerequisite)), []string{"Yes", "No"})
		if err != nil {
			return err
		}
		if answer != "Yes" {
			return fmt.Errorf("Package `%s` can't be installed without %s", pkg.Name, prerequisite.Name)
		}
	}

	// The nodes go first, as the package providing the prerequisite can rely on them too
	if prerequisite.Talos != nil {
		if err := runner.prerequisiteService.PrepareNodes(*prerequisite.Talos, cluster); err != nil {
			return err
		}
	}

	if prerequisite.Package != "" {
		if err := runner.ensureFor(provider, requiredBy); err != nil {
			return err
		}
		return runner.install(provider)
	}

	// Without a package to install, the prerequisite has to be met now
	missing, err = runner.prerequisiteService.Check(prerequisite, cluster)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return fmt.Errorf("Package `%s` still needs %s after setting it up, as the cluster is missing %s", pkg.Name, prerequisite.Name, strings.Join(missing, ", "))
	}

	return nil
}

func describeFix(prerequisite models.ChartPrerequisite) string {
	var steps []string
	if prerequisite.Talos != nil {
		steps = append(steps, "bbe updates each node that needs it, and nodes missing an extension are upgraded and restart one at a time")
	}
	if prerequisite.Package != "" {
		steps = append(steps, fmt.Sprintf("bbe installs the `%s` package", prerequisite.Package))
	}

	return strings.Join(steps, ", then ")
}

func (runner *prerequisiteRunner) clusterAccess() (models.ClusterAccess, error) {
	if runner.cluster != nil {
		return *runner.cluster, nil
	}

	controlPlaneIp, err := runner.talosService.GetControlPlaneIp(runner.helperService, constants.ControlplaneConfigFile)
	if err != nil {
		return models.ClusterAccess{}, fmt.Errorf("Failed to find the cluster's control plane: %w", err)
	}

	runner.cluster = &models.ClusterAccess{
		KubeContext:    runner.bbeConfig.Bbe.Cluster.Context,
		ControlPlaneIp: controlPlaneIp,
		TalosConfig:    runner.helperService.GetConfigFilePath(constants.TalosConfigFile),
	}

	return *runner.cluster, nil
}
