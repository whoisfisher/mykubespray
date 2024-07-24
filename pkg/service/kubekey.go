package service

import "github.com/xiaoming/offline-kubespray/pkg/utils"

type KubekeyService interface {
	GenerateConfig() error
	CreateCluster(logChan chan utils.LogEntry) error
	DeleteCluster(logChan chan utils.LogEntry) error
	AddNode(logChan chan utils.LogEntry) error
	DeleteNode(nodeName string, logChan chan utils.LogEntry) error
}
