package cmd

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/config_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/helm_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/helper_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/kubernetes_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/package_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/ui_service"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:     "uninstall [package...]",
	Aliases: []string{"remove"},
	Short:   "Uninstall BBE packages, deleting everything they keep in the cluster",
	Run: func(cmd *cobra.Command, args []string) {
		helperService := helper_service.HelperService{}
		uiService := ui_service.UiService{}
		configService := config_service.ConfigService{}
		packageService := package_service.PackageService{}
		helmService := helm_service.HelmService{}
		kubernetesService := kubernetes_service.KubernetesService{}

		err := uninstallCommand(args, helperService, uiService, configService, packageService, helmService, kubernetesService)
		if err != nil {
			logger.Error("", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}

func uninstallCommand(args []string, helperService interfaces.HelperServiceInterface, uiService interfaces.UiServiceInterface, configService interfaces.ConfigServiceInterface, packageService interfaces.PackageServiceInterface, helmService interfaces.HelmServiceInterface, kubernetesService interfaces.KubernetesServiceInterface) error {
	bbeConfig, err := configService.GetBbeConfig(helperService)
	if err != nil || bbeConfig.Bbe.Cluster.Name == "" {
		return errors.New("No BBE cluster found, please run 'bbe setup' to create your cluster")
	}
	if len(bbeConfig.Bbe.Packages) == 0 {
		return errors.New("No packages are installed")
	}

	names := args
	if len(names) == 0 {
		installed := []string{}
		for _, pkg := range bbeConfig.Bbe.Packages {
			installed = append(installed, pkg.Name)
		}
		names, err = uiService.CreateMultiChoose("Select packages to uninstall", installed, nil)
		if err != nil {
			return err
		}
	}

	library, err := packageService.GetAll()
	if err != nil {
		return err
	}

	remover := &packageRemover{
		helperService:     helperService,
		uiService:         uiService,
		configService:     configService,
		helmService:       helmService,
		kubernetesService: kubernetesService,
		bbeConfig:         bbeConfig,
		library:           library,
		charts:            map[string]*models.HelmChart{},
	}

	return remover.remove(names)
}

// Uninstalls packages once confirmed, deleting their namespaces with everything in them, and refuses to remove a
// package another installed one needs
type packageRemover struct {
	helperService     interfaces.HelperServiceInterface
	uiService         interfaces.UiServiceInterface
	configService     interfaces.ConfigServiceInterface
	helmService       interfaces.HelmServiceInterface
	kubernetesService interfaces.KubernetesServiceInterface
	bbeConfig         *models.BbeConfig
	library           []models.ChartEntry
	// Each package's chart at its installed version, nil when it can't be read
	charts map[string]*models.HelmChart
}

func (remover *packageRemover) remove(names []string) error {
	for _, name := range names {
		if !slices.ContainsFunc(remover.bbeConfig.Bbe.Packages, func(pkg models.LocalPackage) bool { return pkg.Name == name }) {
			return fmt.Errorf("Package `%s` isn't installed", name)
		}
	}

	for _, name := range names {
		if dependents := remover.dependents(name, names); len(dependents) > 0 {
			return fmt.Errorf("Package `%s` can't be uninstalled, as %s needs it. Uninstall %s first, or together with it", name, describeNames(dependents), describeNames(dependents))
		}
	}

	for _, name := range remover.order(names) {
		chart := remover.chart(name)

		confirmed, err := remover.confirm(name, chart)
		if err != nil {
			return err
		}
		if !confirmed {
			logger.Info(fmt.Sprintf("Package `%s` is kept", name))
			continue
		}

		if err := remover.uninstall(name, chart); err != nil {
			return err
		}

		remover.bbeConfig.Bbe.Packages = slices.DeleteFunc(remover.bbeConfig.Bbe.Packages, func(pkg models.LocalPackage) bool { return pkg.Name == name })
		if err := remover.configService.UpdateBbePackages(remover.helperService, remover.bbeConfig.Bbe.Packages); err != nil {
			return fmt.Errorf("Failed to update BBE configuration: %w", err)
		}
		logger.Info(fmt.Sprintf("Package `%s` is uninstalled", name))
	}

	return nil
}

// The installed packages that aren't being removed and need this one, as their charts' prerequisites name it
func (remover *packageRemover) dependents(name string, removing []string) []string {
	dependents := []string{}
	for _, pkg := range remover.bbeConfig.Bbe.Packages {
		if slices.Contains(removing, pkg.Name) {
			continue
		}

		chart := remover.chart(pkg.Name)
		if chart != nil && slices.ContainsFunc(chart.Prerequisites, func(prerequisite models.ChartPrerequisite) bool { return prerequisite.Package == name }) {
			dependents = append(dependents, pkg.Name)
		}
	}

	return dependents
}

// Packages others being removed need go after them, so nothing loses what it runs on while it's still installed
func (remover *packageRemover) order(names []string) []string {
	needed := func(name string) bool {
		return slices.ContainsFunc(names, func(other string) bool {
			chart := remover.chart(other)
			return other != name && chart != nil && slices.ContainsFunc(chart.Prerequisites, func(prerequisite models.ChartPrerequisite) bool { return prerequisite.Package == name })
		})
	}

	ordered := []string{}
	for _, name := range names {
		if !needed(name) {
			ordered = append(ordered, name)
		}
	}
	for _, name := range names {
		if needed(name) {
			ordered = append(ordered, name)
		}
	}

	return ordered
}

// The package's chart at its installed version, to read what it needs and what else uninstalling it deletes
func (remover *packageRemover) chart(name string) *models.HelmChart {
	if chart, read := remover.charts[name]; read {
		return chart
	}
	remover.charts[name] = nil

	index := slices.IndexFunc(remover.bbeConfig.Bbe.Packages, func(pkg models.LocalPackage) bool { return pkg.Name == name })
	libraryIndex := slices.IndexFunc(remover.library, func(chart models.ChartEntry) bool { return chart.Name == name })
	if index < 0 || libraryIndex < 0 {
		logger.Warning(fmt.Sprintf("Package `%s` isn't in the package library, so bbe can't tell what it needs or keeps", name))
		return nil
	}

	entry := remover.library[libraryIndex]
	chart, err := remover.helmService.PullChart(entry.RepositoryUrl, name, remover.bbeConfig.Bbe.Packages[index].Version)
	if err != nil {
		logger.Warning(fmt.Sprintf("Package `%s`'s chart can't be read, so bbe can't tell what it needs or keeps: %v", name, err))
		return nil
	}
	remover.charts[name] = chart

	return chart
}

func namespacesOf(name string, chart *models.HelmChart) []string {
	namespaces := []string{name}
	if chart != nil && chart.Uninstall != nil {
		for _, namespace := range chart.Uninstall.Namespaces {
			if !slices.Contains(namespaces, namespace) {
				namespaces = append(namespaces, namespace)
			}
		}
	}

	return namespaces
}

// Lists what uninstalling deletes and keeps, and asks, defaulting to no
func (remover *packageRemover) confirm(name string, chart *models.HelmChart) (bool, error) {
	lines := []string{fmt.Sprintf("Uninstalling `%s` deletes its release, and with it:", name)}
	for _, namespace := range namespacesOf(name, chart) {
		volumes, exists, err := remover.kubernetesService.NamespaceVolumes(remover.bbeConfig.Bbe.Cluster.Context, namespace)
		if err != nil {
			return false, err
		}
		if !exists {
			continue
		}

		line := fmt.Sprintf("  - the namespace `%s` and everything in it, such as settings, databases and credentials", namespace)
		if len(volumes) > 0 {
			described := []string{}
			for _, volume := range volumes {
				described = append(described, fmt.Sprintf("%s (%s)", volume.Name, volume.Size))
			}
			line += fmt.Sprintf(", and its %d volumes: %s", len(volumes), strings.Join(described, ", "))
		}
		lines = append(lines, line)
	}
	if chart != nil && chart.Uninstall != nil && len(chart.Uninstall.Keeps) > 0 {
		lines = append(lines, "It keeps:")
		for _, kept := range chart.Uninstall.Keeps {
			lines = append(lines, "  - "+kept)
		}
	}
	logger.Warning(strings.Join(lines, "\n"))

	answer, err := remover.uiService.CreateSelect(fmt.Sprintf("Uninstall `%s` and delete all of this?", name), []string{"No", "Yes"})
	if err != nil {
		return false, err
	}

	return answer == "Yes", nil
}

func (remover *packageRemover) uninstall(name string, chart *models.HelmChart) error {
	context := remover.bbeConfig.Bbe.Cluster.Context

	if chart != nil && chart.Uninstall != nil {
		for _, patch := range chart.Uninstall.Patches {
			if err := remover.kubernetesService.Patch(context, patch.Namespace, patch.Resource, patch.Name, patch.Patch); err != nil {
				return err
			}
		}
	}

	// A release that failed to install, or was removed by hand, still leaves its namespace to delete
	if remover.helmService.IsPackageInstalled(name, name, context) {
		if err := remover.helmService.UninstallChart(name, name, context); err != nil {
			return err
		}
	}

	for _, namespace := range namespacesOf(name, chart) {
		if err := remover.kubernetesService.DeleteNamespace(context, namespace); err != nil {
			return err
		}
	}

	return nil
}

func describeNames(names []string) string {
	quoted := []string{}
	for _, name := range names {
		quoted = append(quoted, "`"+name+"`")
	}

	return strings.Join(quoted, " and ")
}
