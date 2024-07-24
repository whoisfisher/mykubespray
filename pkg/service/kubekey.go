package service

import "github.com/whoisfisher/mykubespray/pkg/utils"

type KubekeyService interface {
	GenerateConfig() error
	CreateCluster(logChan chan utils.LogEntry) error
	DeleteCluster(logChan chan utils.LogEntry) error
	AddNode(logChan chan utils.LogEntry) error
	DeleteNode(nodeName string, logChan chan utils.LogEntry) error
}
