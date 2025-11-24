package helm

import (
	"context"
	"errors"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	"github.com/aws/smithy-go/ptr"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	helmVals "helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	helmRelease "helm.sh/helm/v3/pkg/release"
	helmDriver "helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	capiaddons "sigs.k8s.io/cluster-api-addon-provider-helm/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Client struct {
	actionConfig *action.Configuration
	settings     *cli.EnvSettings
	namespace    string
}

func NewClient(kubeconfig []byte, namespace string) (*Client, error) {
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		panic(err)
	}

	settings := cli.New()
	actionConfig := new(action.Configuration)

	// Create a RESTClientGetter
	getter := genericclioptions.NewConfigFlags(false)
	getter.APIServer = &restConfig.Host
	getter.BearerToken = &restConfig.BearerToken
	getter.Namespace = &namespace
	getter.Insecure = ptr.Bool(true)
	getter.CAFile = &restConfig.CAFile

	wrapper := func(*rest.Config) *rest.Config {
		return restConfig
	}
	getter.WithWrapConfigFn(wrapper)

	// Initialize the action configuration
	if err := actionConfig.Init(getter,
		namespace,
		"secret",
		func(format string, v ...interface{}) {}); err != nil {
		return nil, err
	}

	return &Client{
		actionConfig: actionConfig,
		settings:     settings,
		namespace:    namespace,
	}, nil

}

func (c *Client) InstallOrUpgradeChart(ctx context.Context, spec talosv1alpha1.HelmSpec) (*helmRelease.Release, error) {

	logger := log.FromContext(ctx)

	_, err := c.GetRelease(ctx, spec.ReleaseName)
	if err != nil {
		if errors.Is(err, helmDriver.ErrReleaseNotFound) {
			// Install the chart
			logger.Info("Installing new helm chart", "releaseName", spec.ReleaseName)
			return c.installChart(ctx, spec)
		}
		return nil, err
	}
	// Upgrade the chart
	return c.upgradeChart(ctx, spec)
}

func (c *Client) upgradeChart(ctx context.Context, spec talosv1alpha1.HelmSpec) (*helmRelease.Release, error) {
	logger := log.FromContext(ctx)
	logger.Info("Upgrading existing helm chart", "releaseName", spec.ReleaseName)

	upgradeClient := generateHelmUpgradeConfig(c.actionConfig, spec.Options)
	upgradeClient.RepoURL = spec.RepoURL
	upgradeClient.Version = spec.Version
	upgradeClient.Namespace = spec.ReleaseNamespace

	// Locate chart
	chartPath, err := upgradeClient.LocateChart(spec.ChartName, c.settings)
	if err != nil {
		return nil, err
	}
	// Load chart
	chartReq, err := loader.Load(chartPath)
	if err != nil {
		return nil, err
	}
	vals, err := handleValuesTemplate(spec.ValuesTemplate)
	if err != nil {
		return nil, err
	}
	// Upgrade chart
	return upgradeClient.RunWithContext(ctx, spec.ReleaseName, chartReq, vals)
}

func (c *Client) GetRelease(ctx context.Context, releaseName string) (*helmRelease.Release, error) {
	getAction := action.NewGet(c.actionConfig)
	release, err := getAction.Run(releaseName)
	if err != nil {
		return nil, err
	}
	return release, nil
}

func (c *Client) installChart(ctx context.Context, spec talosv1alpha1.HelmSpec) (*helmRelease.Release, error) {

	// TODO: Handle authentication using spec.Credentials and spec.TLSConfig

	installConfig := generateHelmInstallConfig(c.actionConfig, spec.Options)
	installConfig.RepoURL = spec.RepoURL
	installConfig.Version = spec.Version
	installConfig.Namespace = spec.ReleaseNamespace
	var err error

	if spec.ReleaseName == "" {
		installConfig.GenerateName = true
		spec.ReleaseName, _, err = installConfig.NameAndChart([]string{spec.ChartName})
		if err != nil {
			return nil, err
		}
	}
	installConfig.ReleaseName = spec.ReleaseName

	// Locate chart
	chartPath, err := installConfig.LocateChart(spec.ChartName, c.settings)
	if err != nil {
		return nil, err
	}
	// Load chart
	chartReq, err := loader.Load(chartPath)
	if err != nil {
		return nil, err
	}
	vals, err := handleValuesTemplate(spec.ValuesTemplate)
	if err != nil {
		return nil, err
	}

	// Install chart
	return installConfig.RunWithContext(ctx, chartReq, vals)
}

func (c *Client) UninstallChart(ctx context.Context, releaseName string) (
	*helmRelease.UninstallReleaseResponse, error) {

	uninstallClient := action.NewUninstall(c.actionConfig)
	_, err := uninstallClient.Run(releaseName)
	if err != nil {
		return nil, err
	}

	return uninstallClient.Run(releaseName)

}

func handleValuesTemplate(valuesTemplate string) (map[string]interface{}, error) {
	if valuesTemplate == "" {
		return map[string]interface{}{}, nil
	}
	// Read the values template and parse it
	valueOpts := &helmVals.Options{
		Values: []string{valuesTemplate},
	}
	return valueOpts.MergeValues(getter.All(cli.New()))
}

func generateHelmInstallConfig(actionConfig *action.Configuration,
	helmOptions *capiaddons.HelmOptions) *action.Install {
	installClient := action.NewInstall(actionConfig)
	installClient.CreateNamespace = true
	if actionConfig.RegistryClient != nil {
		installClient.SetRegistryClient(actionConfig.RegistryClient)
	}
	if helmOptions == nil {
		return installClient
	}

	installClient.DisableHooks = helmOptions.DisableHooks
	installClient.Wait = helmOptions.Wait
	installClient.WaitForJobs = helmOptions.WaitForJobs
	if helmOptions.Timeout != nil {
		installClient.Timeout = helmOptions.Timeout.Duration
	}
	installClient.SkipCRDs = helmOptions.SkipCRDs
	installClient.SubNotes = helmOptions.SubNotes
	installClient.DisableOpenAPIValidation = helmOptions.DisableOpenAPIValidation
	installClient.Atomic = helmOptions.Atomic
	installClient.IncludeCRDs = helmOptions.Install.IncludeCRDs
	installClient.CreateNamespace = helmOptions.Install.CreateNamespace

	return installClient
}

func generateHelmUpgradeConfig(actionConfig *action.Configuration,
	helmOptions *capiaddons.HelmOptions) *action.Upgrade {
	upgradeClient := action.NewUpgrade(actionConfig)
	if actionConfig.RegistryClient != nil {
		upgradeClient.SetRegistryClient(actionConfig.RegistryClient)
	}
	if helmOptions == nil {
		return upgradeClient
	}

	upgradeClient.DisableHooks = helmOptions.DisableHooks
	upgradeClient.Wait = helmOptions.Wait
	upgradeClient.WaitForJobs = helmOptions.WaitForJobs
	if helmOptions.Timeout != nil {
		upgradeClient.Timeout = helmOptions.Timeout.Duration
	}
	upgradeClient.SkipCRDs = helmOptions.SkipCRDs
	upgradeClient.SubNotes = helmOptions.SubNotes
	upgradeClient.DisableOpenAPIValidation = helmOptions.DisableOpenAPIValidation
	upgradeClient.Atomic = helmOptions.Atomic
	upgradeClient.Force = helmOptions.Upgrade.Force
	upgradeClient.ResetValues = helmOptions.Upgrade.ResetValues
	upgradeClient.ReuseValues = helmOptions.Upgrade.ReuseValues
	upgradeClient.ResetThenReuseValues = helmOptions.Upgrade.ResetThenReuseValues
	upgradeClient.MaxHistory = helmOptions.Upgrade.MaxHistory
	upgradeClient.CleanupOnFail = helmOptions.Upgrade.CleanupOnFail

	return upgradeClient
}
