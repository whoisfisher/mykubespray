package kubernetes

import (
	"context"
	"fmt"
	"github.com/ghodss/yaml"
	helm "github.com/mittwald/go-helm-client"
	"github.com/whoisfisher/mykubespray/pkg/entity"
	"github.com/whoisfisher/mykubespray/pkg/logger"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	"sync"
)

func (client *K8sClient) ApplyYAML(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		logger.GetLogger().Printf("Error reading file %s: %s", file, err)
		return err
	}

	var unstructuredData *unstructured.Unstructured
	if err := yaml.Unmarshal(data, &unstructuredData); err != nil {
		logger.GetLogger().Printf("Error unmarshalling YAML: %s", err)
		return err
	}
	obj := unstructuredData.Object

	apiVersion, kind := obj["apiVersion"].(string), obj["kind"].(string)
	if apiVersion == "" || kind == "" {
		err := fmt.Errorf("apiVersion or kind not found in YAML")
		logger.GetLogger().Println(err)
		return err
	}

	gvr, namespaced := getGVR(kind, apiVersion)
	if gvr == (schema.GroupVersionResource{}) {
		err := fmt.Errorf("unsupported kind: %s", kind)
		logger.GetLogger().Println(err)
		return err
	}

	resourceClient := client.DynamicClient.Resource(gvr)
	namespace := getNamespace(obj)

	if namespaced && namespace != "" {
		return applyInNamespace(resourceClient, namespace, unstructuredData)
	}

	return applyNonNamespaced(resourceClient, unstructuredData)
}

func (client *K8sClient) ApplyYAMLs(files []string) (*entity.ApplyResults, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	var results []entity.SingleApplyResult
	var overallSuccess = true

	for _, file := range files {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			var result entity.SingleApplyResult
			result.FileName = file
			if err := client.ApplyYAML(file); err != nil {
				result.Success = false
				result.Error = err.Error()
				mu.Lock()
				overallSuccess = false
				mu.Unlock()
			} else {
				result.Success = true
			}
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(file)
	}

	wg.Wait()
	return &entity.ApplyResults{
		OverallSuccess: overallSuccess,
		Results:        results,
	}, nil
}

func (client *K8sClient) DeployYAML(content string) error {
	var unstructuredData *unstructured.Unstructured
	if err := yaml.Unmarshal([]byte(content), &unstructuredData); err != nil {
		logger.GetLogger().Printf("Error unmarshalling YAML: %s", err)
		return err
	}
	obj := unstructuredData.Object

	apiVersion, kind := obj["apiVersion"].(string), obj["kind"].(string)
	if apiVersion == "" || kind == "" {
		err := fmt.Errorf("apiVersion or kind not found in YAML")
		logger.GetLogger().Println(err)
		return err
	}

	gvr, namespaced := getGVR(kind, apiVersion)
	if gvr == (schema.GroupVersionResource{}) {
		err := fmt.Errorf("unsupported kind: %s", kind)
		logger.GetLogger().Println(err)
		return err
	}

	resourceClient := client.DynamicClient.Resource(gvr)
	namespace := getNamespace(obj)

	if namespaced && namespace != "" {
		return applyInNamespace(resourceClient, namespace, unstructuredData)
	}

	return applyNonNamespaced(resourceClient, unstructuredData)
}

func (client *K8sClient) DeployYAMLs(contents []string) (*entity.ApplyResults, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	var results []entity.SingleApplyResult
	var overallSuccess = true

	for _, content := range contents {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			var result entity.SingleApplyResult
			result.FileName = file
			if err := client.DeployYAML(content); err != nil {
				result.Success = false
				result.Error = err.Error()
				mu.Lock()
				overallSuccess = false
				mu.Unlock()
			} else {
				result.Success = true
			}
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(content)
	}

	wg.Wait()
	return &entity.ApplyResults{
		OverallSuccess: overallSuccess,
		Results:        results,
	}, nil
}

func (client *K8sClient) AddOrUpdateChartRepo(helmRepo entity.HelmRepository) error {
	var repoEntry repo.Entry
	repoEntry.Name = helmRepo.Name
	repoEntry.URL = helmRepo.Url
	repoEntry.Username = helmRepo.Username
	repoEntry.Password = helmRepo.Password
	repoEntry.CertFile = helmRepo.CertFile
	repoEntry.CAFile = helmRepo.CAFile
	repoEntry.KeyFile = helmRepo.KeyFile
	repoEntry.InsecureSkipTLSverify = helmRepo.InsecureSkipTlsVerify
	if err := client.HelmClient.AddOrUpdateChartRepo(repoEntry); err != nil {
		logger.GetLogger().Errorf("Faile to add or update helm repository to cluster: %s", err.Error())
		return err
	}
	return nil
}

func (client *K8sClient) InstallOrUpgradeChart(info entity.HelmChartInfo) (*release.Release, error) {
	chartSpec := helm.ChartSpec{
		ReleaseName:     info.ReleaseName,
		ChartName:       info.ChartName,
		Namespace:       info.Namespace,
		ValuesYaml:      info.ValuesYaml,
		CreateNamespace: info.CreateNamespace,
	}
	release1, err := client.HelmClient.InstallOrUpgradeChart(context.TODO(), &chartSpec, &helm.GenericHelmOptions{})
	if err != nil {
		logger.GetLogger().Errorf("Faile to install or update chart release: %s", err.Error())
		return nil, err
	}
	return release1, nil
}

func (client *K8sClient) ListDeployedReleases(info entity.HelmChartInfo) ([]*release.Release, error) {
	releases, err := client.HelmClient.ListDeployedReleases()
	if err != nil {
		logger.GetLogger().Errorf("Faile to list deployed helm chart release: %s", err.Error())
		return nil, err
	}
	return releases, nil
}

func (client *K8sClient) UninstallRelease(info entity.HelmChartInfo) error {
	chartSpec := helm.ChartSpec{
		ReleaseName:     info.ReleaseName,
		ChartName:       info.ChartName,
		Namespace:       info.Namespace,
		ValuesYaml:      info.ValuesYaml,
		CreateNamespace: info.CreateNamespace,
	}
	err := client.HelmClient.UninstallRelease(&chartSpec)
	if err != nil {
		logger.GetLogger().Errorf("Faile to uninstall deployed helm chart release[%s]: %s", info.ReleaseName, err.Error())
		return err
	}
	return nil
}
