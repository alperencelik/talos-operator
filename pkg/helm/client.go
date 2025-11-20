/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package helm

import (
	"fmt"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/client-go/tools/clientcmd"
)

// Client provides methods to interact with Helm
type Client struct {
	actionConfig *action.Configuration
	settings     *cli.EnvSettings
	namespace    string
}

// NewClient creates a new Helm client
func NewClient(kubeconfig []byte, namespace string) (*Client, error) {
	if namespace == "" {
		namespace = "default"
	}

	// Create a temporary file for kubeconfig
	tmpFile, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary kubeconfig file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(kubeconfig); err != nil {
		return nil, fmt.Errorf("failed to write kubeconfig: %w", err)
	}
	tmpFile.Close()

	// Create REST config from kubeconfig
	restConfig, err := clientcmd.BuildConfigFromFlags("", tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to build REST config: %w", err)
	}

	// Create action configuration
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(
		&genericRESTClientGetter{
			restConfig: restConfig,
			namespace:  namespace,
		},
		namespace,
		"secret",
		func(format string, v ...interface{}) {
			// Logger - can be customized
		},
	); err != nil {
		return nil, fmt.Errorf("failed to initialize Helm action config: %w", err)
	}

	settings := cli.New()

	return &Client{
		actionConfig: actionConfig,
		settings:     settings,
		namespace:    namespace,
	}, nil
}

// InstallOrUpgrade installs or upgrades a Helm chart
func (c *Client) InstallOrUpgrade(
	releaseName, repoURL, chartName, version string,
	values map[string]interface{},
) (int, error) {
	// Check if release exists
	histClient := action.NewHistory(c.actionConfig)
	histClient.Max = 1
	if _, err := histClient.Run(releaseName); err != nil {
		// Release doesn't exist, install it
		return c.install(releaseName, repoURL, chartName, version, values)
	}

	// Release exists, upgrade it
	return c.upgrade(releaseName, repoURL, chartName, version, values)
}

// install installs a Helm chart
func (c *Client) install(
	releaseName, repoURL, chartName, version string,
	values map[string]interface{},
) (int, error) {
	client := action.NewInstall(c.actionConfig)
	client.ReleaseName = releaseName
	client.Namespace = c.namespace
	client.CreateNamespace = true
	client.Wait = true
	client.Timeout = 0
	if version != "" {
		client.Version = version
	}

	// Locate chart
	chartPath, err := c.locateChart(repoURL, chartName, version)
	if err != nil {
		return 0, fmt.Errorf("failed to locate chart: %w", err)
	}

	// Load chart
	chartRequested, err := loader.Load(chartPath)
	if err != nil {
		return 0, fmt.Errorf("failed to load chart: %w", err)
	}

	// Install chart
	release, err := client.Run(chartRequested, values)
	if err != nil {
		return 0, fmt.Errorf("failed to install chart: %w", err)
	}

	return release.Version, nil
}

// upgrade upgrades a Helm chart
func (c *Client) upgrade(
	releaseName, repoURL, chartName, version string,
	values map[string]interface{},
) (int, error) {
	client := action.NewUpgrade(c.actionConfig)
	client.Namespace = c.namespace
	client.Wait = true
	client.Timeout = 0
	if version != "" {
		client.Version = version
	}

	// Locate chart
	chartPath, err := c.locateChart(repoURL, chartName, version)
	if err != nil {
		return 0, fmt.Errorf("failed to locate chart: %w", err)
	}

	// Load chart
	chartRequested, err := loader.Load(chartPath)
	if err != nil {
		return 0, fmt.Errorf("failed to load chart: %w", err)
	}

	// Upgrade chart
	release, err := client.Run(releaseName, chartRequested, values)
	if err != nil {
		return 0, fmt.Errorf("failed to upgrade chart: %w", err)
	}

	return release.Version, nil
}

// Uninstall uninstalls a Helm release
func (c *Client) Uninstall(releaseName string) error {
	client := action.NewUninstall(c.actionConfig)
	client.Wait = true
	client.Timeout = 0

	if _, err := client.Run(releaseName); err != nil {
		return fmt.Errorf("failed to uninstall release: %w", err)
	}

	return nil
}

// locateChart locates a chart from a repository
func (c *Client) locateChart(repoURL, chartName, version string) (string, error) {
	// Add repository
	repoEntry := &repo.Entry{
		Name: "addon-repo",
		URL:  repoURL,
	}

	chartRepo, err := repo.NewChartRepository(repoEntry, getter.All(c.settings))
	if err != nil {
		return "", fmt.Errorf("failed to create chart repository: %w", err)
	}

	// Download index
	indexFile, err := chartRepo.DownloadIndexFile()
	if err != nil {
		return "", fmt.Errorf("failed to download repository index: %w", err)
	}

	// Load index
	index, err := repo.LoadIndexFile(indexFile)
	if err != nil {
		return "", fmt.Errorf("failed to load repository index: %w", err)
	}

	// Get chart version
	chartVersion, err := index.Get(chartName, version)
	if err != nil {
		return "", fmt.Errorf("failed to get chart version: %w", err)
	}

	if len(chartVersion.URLs) == 0 {
		return "", fmt.Errorf("chart has no downloadable URLs")
	}

	// Download chart
	destDir, err := os.MkdirTemp("", "helm-chart-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	client := action.NewPullWithOpts(action.WithConfig(c.actionConfig))
	client.DestDir = destDir
	client.RepoURL = repoURL
	if version != "" {
		client.Version = version
	}

	chartPath, err := client.Run(chartName)
	if err != nil {
		return "", fmt.Errorf("failed to pull chart: %w", err)
	}

	return chartPath, nil
}
