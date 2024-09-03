package entity

import "github.com/whoisfisher/mykubespray/pkg/utils/kubernetes"

type KubernetesFilesConf struct {
	kubernetes.K8sConfig
	Files []string
}
