package runtime_provider

import (
	"context"
	auth "github.com/deislabs/oras/pkg/auth/docker"
	"helm.sh/helm/pkg/action"
	"helm.sh/helm/pkg/chart"
	"helm.sh/helm/pkg/chartutil"
	"helm.sh/helm/pkg/registry"
	"helm.sh/helm/pkg/release"
	"path/filepath"
	"testing"
)


func Install() (*release.Release, error) {
	cfg := newActionConfig(false)
	credentialsFile := filepath.Join(settings.Home.Registry(), registry.CredentialsFileBasename)
	client, err := auth.NewClient(credentialsFile)
	if err != nil {
		panic(err)
	}
	resolver, err := client.Resolver(context.Background())
	if err != nil {
		panic(err)
	}
	cfg.RegistryClient = registry.NewClient(&registry.ClientOptions{
		Debug: settings.Debug,
		Out:   nil,
		Authorizer: registry.Authorizer{
			Client: client,
		},
		Resolver: registry.Resolver{
			Resolver: resolver,
		},
		CacheRootDir: settings.Home.Registry(),
	})

	chartReq := &chart.Chart{}

	rlsName := ""
	installClient := action.NewInstall(cfg)
	installClient.ReleaseName = rlsName

	validInstallableChart, err := chartutil.IsChartInstallable(chartReq)
	if !validInstallableChart {
		return nil, err
	}
	installClient.Namespace = getNamespace()
	return installClient.Run(chartReq)
}

func UnInstall(rlsName string) (*release.UninstallReleaseResponse, error) {
	cfg := newActionConfig(false)

	client := action.NewUninstall(cfg)
	return client.Run(rlsName)
}

func TestAcitonConfig(t *testing.T){

}
