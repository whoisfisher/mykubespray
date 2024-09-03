package kubernetes

import (
	"fmt"
	"github.com/whoisfisher/mykubespray/pkg/logger"
	"gopkg.in/yaml.v2"
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

	var obj map[string]interface{}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		logger.GetLogger().Printf("Error unmarshalling YAML: %s", err)
		return err
	}

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
		return applyInNamespace(resourceClient, namespace, obj)
	}

	return applyNonNamespaced(resourceClient, obj)
}

type SingleApplyResult struct {
	FileName string
	Success  bool
	Error    string
}

type ApplyResults struct {
	OverallSuccess bool
	Results        []SingleApplyResult
}

func (client *K8sClient) ApplyYAMLs(files []string) (*ApplyResults, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	var results []SingleApplyResult
	var overallSuccess = true

	for _, file := range files {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			var result SingleApplyResult
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
	return &ApplyResults{
		OverallSuccess: overallSuccess,
		Results:        results,
	}, nil
}
