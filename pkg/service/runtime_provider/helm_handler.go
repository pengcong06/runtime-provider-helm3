// Copyright 2018 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package runtime_provider

import (
	"context"
	"fmt"
	"regexp"

	"helm.sh/helm/pkg/action"
	"helm.sh/helm/pkg/chart"
	rls "helm.sh/helm/pkg/release"

	"google.golang.org/grpc/transport"

	runtimeclient "openpitrix.io/openpitrix/pkg/client/runtime"
	"openpitrix.io/openpitrix/pkg/constants"
	"openpitrix.io/openpitrix/pkg/gerr"
	"openpitrix.io/openpitrix/pkg/util/funcutil"
)

var (
	ClusterNameReg    = `^[a-z]([-a-z0-9]*[a-z0-9])?$`
	ClusterNameRegExp = regexp.MustCompile(ClusterNameReg)
)

type HelmHandler struct {
	ctx       context.Context
	RuntimeId string
}

func GetHelmHandler(ctx context.Context, runtimeId string) *HelmHandler {
	helmHandler := new(HelmHandler)
	helmHandler.ctx = ctx
	helmHandler.RuntimeId = runtimeId
	return helmHandler
}

//func (p *HelmHandler) initKubeClient() (*kubernetes.Clientset, *rest.Config, error) {
//	kubeconfigGetter := func() (*clientcmdapi.Config, error) {
//		runtime, err := runtimeclient.NewRuntime(p.ctx, p.RuntimeId)
//		if err != nil {
//			return nil, err
//		}
//
//		return clientcmd.Load([]byte(runtime.RuntimeCredentialContent))
//	}
//
//	config, err := clientcmd.BuildConfigFromKubeconfigGetter("", kubeconfigGetter)
//	if err != nil {
//		return nil, nil, err
//	}
//
//	config.CAData = config.CAData[0:0]
//	config.TLSClientConfig.Insecure = true
//
//	clientset, err := kubernetes.NewForConfig(config)
//	if err != nil {
//		return nil, nil, err
//	}
//	return clientset, config, err
//}

func (p *HelmHandler) InstallReleaseFromChart(c *chart.Chart, ns string, rawVals []byte, releaseName string) error {
	runtime, err := runtimeclient.NewRuntime(p.ctx, p.RuntimeId)
	if err != nil {
		return err
	}

	cfg := NewActionConfig(false, []byte(runtime.RuntimeCredentialContent))

	installClient := action.NewInstall(cfg)
	//installClient.ValueOptions.StringValues = []string{}
	installClient.ReleaseName = releaseName

	//validInstallableChart, err := chartutil.IsChartInstallable(c)
	//if !validInstallableChart {
	//	return err
	//}
	installClient.Namespace = getNamespace([]byte(runtime.RuntimeCredentialContent))
	_, err = installClient.Run(c, nil)
	return err
}

func (p *HelmHandler) UpdateReleaseFromChart(releaseName string, c *chart.Chart, rawVals []byte) error {
	runtime, err := runtimeclient.NewRuntime(p.ctx, p.RuntimeId)
	if err != nil {
		return err
	}
	cfg := NewActionConfig(false, []byte(runtime.RuntimeCredentialContent))
	chartReq := &chart.Chart{}

	updateClient := action.NewUpgrade(cfg)

	//validInstallableChart, err := chartutil.IsChartInstallable(chartReq)
	//if !validInstallableChart {
	//	return err
	//}
	updateClient.Namespace = getNamespace([]byte(runtime.RuntimeCredentialContent))
	_, err = updateClient.Run(releaseName, chartReq, nil)
	return err
}

func (p *HelmHandler) RollbackRelease(releaseName string) error {
	runtime, err := runtimeclient.NewRuntime(p.ctx, p.RuntimeId)
	if err != nil {
		return err
	}
	cfg := NewActionConfig(false, []byte(runtime.RuntimeCredentialContent))

	rollbackClient := action.NewRollback(cfg)

	err = rollbackClient.Run(releaseName)

	return err
}

func (p *HelmHandler) DeleteRelease(releaseName string, purge bool) error {
	runtime, err := runtimeclient.NewRuntime(p.ctx, p.RuntimeId)
	if err != nil {
		return err
	}
	cfg := NewActionConfig(false, []byte(runtime.RuntimeCredentialContent))

	uninstallClient := action.NewUninstall(cfg)

	_, err = uninstallClient.Run(releaseName)

	return err
}

func (p *HelmHandler) ReleaseStatus(releaseName string) (*rls.Release, error) {
	runtime, err := runtimeclient.NewRuntime(p.ctx, p.RuntimeId)
	if err != nil {
		return nil, err
	}
	cfg := NewActionConfig(false, []byte(runtime.RuntimeCredentialContent))

	statusClient := action.NewStatus(cfg)

	release, err := statusClient.Run(releaseName)
	if err != nil {
		return nil, err
	}

	return release, nil
}

func (p *HelmHandler) CheckClusterNameIsUnique(clusterName string) error {
	if clusterName == "" {
		return fmt.Errorf("cluster name must be provided")
	}

	if !ClusterNameRegExp.MatchString(clusterName) {
		return fmt.Errorf(`cluster name must match with regexp "%s"`, ClusterNameReg)
	}

	// Related to https://github.com/helm/helm/pull/1080
	if len(clusterName) > 14 {
		return fmt.Errorf("the length of config [Name] must be less than 15")
	}

	err := funcutil.WaitForSpecificOrError(func() (bool, error) {
		_, err := p.ReleaseStatus(clusterName)
		if err != nil {
			if _, ok := err.(transport.ConnectionError); ok {
				return false, nil
			}
			return true, nil
		}

		return true, gerr.New(p.ctx, gerr.PermissionDenied, gerr.ErrorHelmReleaseExists, clusterName)
	}, constants.DefaultServiceTimeout, constants.WaitTaskInterval)
	return err
}
