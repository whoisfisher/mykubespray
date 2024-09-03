package service

import (
	"github.com/whoisfisher/mykubespray/pkg/entity"
	"github.com/whoisfisher/mykubespray/pkg/logger"
	"github.com/whoisfisher/mykubespray/pkg/utils/kubernetes"
)

type KubernetesService interface {
	ApplyYAMLs(conf entity.KubernetesFilesConf) *kubernetes.ApplyResults
}

type kubernetesService struct {
}

func NewKubernetesService() kubernetesService {
	return kubernetesService{}
}

func (ks kubernetesService) ApplyYAMLs(conf entity.KubernetesFilesConf) *kubernetes.ApplyResults {
	client, err := kubernetes.NewK8sClient(conf.K8sConfig)
	if err != nil {
		logger.GetLogger().Errorf("Error creating kubernetes client: %v", err)
		return nil
	}
	results, err := client.ApplyYAMLs(conf.Files)
	if err != nil {
		logger.GetLogger().Errorf("Error apply files to kubernetes: %v", err)
		return nil
	}
	return results
}
