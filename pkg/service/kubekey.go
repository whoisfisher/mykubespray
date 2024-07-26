package service

import (
	"github.com/whoisfisher/mykubespray/pkg/entity"
	"github.com/whoisfisher/mykubespray/pkg/utils"
	"log"
)

type KubekeyService interface {
	//GenerateConfig() error
	CreateCluster(conf entity.KubekeyConf, logChan chan utils.LogEntry) error
	DeleteCluster(conf entity.KubekeyConf, logChan chan utils.LogEntry) error
	AddNodeToCluster(conf entity.KubekeyConf, logChan chan utils.LogEntry) error
	DeleteNodeFromCluster(conf entity.KubekeyConf, logChan chan utils.LogEntry) error
}

type kubekeyService struct {
}

func NewKubekeyService() kubekeyService {
	return kubekeyService{}
}

func (ks kubekeyService) CreateCluster(conf entity.KubekeyConf, logChan chan utils.LogEntry) error {
	sshConfig := utils.SSHConfig{}
	for _, host := range conf.Hosts {
		if host.Registry != nil {
			sshConfig.Host = host.Address
			sshConfig.Port = host.Port
			sshConfig.User = host.User
			sshConfig.Password = host.Password
			sshConfig.PrivateKey = host.PrivateKey
			sshConfig.AuthMethods = host.AuthMethods
			conf.Registry = *host.Registry
		}
	}
	connection, err := utils.NewSSHConnection(sshConfig)
	if err != nil {
		log.Fatalf("Failed to create SSH connection: %s", err)
	}
	osCOnf := utils.OSConf{}
	localExecutor := utils.NewLocalExecutor()
	sshExecutor := utils.NewSSHExecutor(*connection)
	osclient := utils.NewOSClient(osCOnf, *sshExecutor, *localExecutor)
	client := utils.NewKubekeyClient(conf, *osclient)
	client.GenerateConfig()
	client.CreateCluster(logChan)
	return nil
}

func (ks kubekeyService) DeleteCluster(conf entity.KubekeyConf, logChan chan utils.LogEntry) error {
	sshConfig := utils.SSHConfig{}
	for _, host := range conf.Hosts {
		if host.Registry != nil {
			sshConfig.Host = host.Address
			sshConfig.Port = host.Port
			sshConfig.User = host.User
			sshConfig.Password = host.Password
			sshConfig.PrivateKey = host.PrivateKey
			sshConfig.AuthMethods = host.AuthMethods
			conf.Registry = *host.Registry
		}
	}
	connection, err := utils.NewSSHConnection(sshConfig)
	if err != nil {
		log.Fatalf("Failed to create SSH connection: %s", err)
	}
	osCOnf := utils.OSConf{}
	localExecutor := utils.NewLocalExecutor()
	sshExecutor := utils.NewSSHExecutor(*connection)
	osclient := utils.NewOSClient(osCOnf, *sshExecutor, *localExecutor)
	client := utils.NewKubekeyClient(conf, *osclient)
	client.GenerateConfig()
	client.DeleteCluster(logChan)
	return nil
}

func (ks kubekeyService) AddNodeToCluster(conf entity.KubekeyConf, logChan chan utils.LogEntry) error {
	sshConfig := utils.SSHConfig{}
	for _, host := range conf.Hosts {
		if host.Registry != nil {
			sshConfig.Host = host.Address
			sshConfig.Port = host.Port
			sshConfig.User = host.User
			sshConfig.Password = host.Password
			sshConfig.PrivateKey = host.PrivateKey
			sshConfig.AuthMethods = host.AuthMethods
			conf.Registry = *host.Registry
		}
	}
	connection, err := utils.NewSSHConnection(sshConfig)
	if err != nil {
		log.Fatalf("Failed to create SSH connection: %s", err)
	}
	osCOnf := utils.OSConf{}
	localExecutor := utils.NewLocalExecutor()
	sshExecutor := utils.NewSSHExecutor(*connection)
	osclient := utils.NewOSClient(osCOnf, *sshExecutor, *localExecutor)
	client := utils.NewKubekeyClient(conf, *osclient)
	client.GenerateConfig()
	client.AddNode(logChan)
	return nil
}

func (ks kubekeyService) DeleteNodeFromCluster(conf entity.KubekeyConf, logChan chan utils.LogEntry) error {
	deleteNode := ""
	sshConfig := utils.SSHConfig{}
	for _, host := range conf.Hosts {
		if host.Registry != nil {
			sshConfig.Host = host.Address
			sshConfig.Port = host.Port
			sshConfig.User = host.User
			sshConfig.Password = host.Password
			sshConfig.PrivateKey = host.PrivateKey
			sshConfig.AuthMethods = host.AuthMethods
			conf.Registry = *host.Registry
		}
		if host.IsDeleted {
			deleteNode = host.Name
		}
	}
	connection, err := utils.NewSSHConnection(sshConfig)
	if err != nil {
		log.Fatalf("Failed to create SSH connection: %s", err)
	}
	osCOnf := utils.OSConf{}
	localExecutor := utils.NewLocalExecutor()
	sshExecutor := utils.NewSSHExecutor(*connection)
	osclient := utils.NewOSClient(osCOnf, *sshExecutor, *localExecutor)
	client := utils.NewKubekeyClient(conf, *osclient)
	client.GenerateConfig()
	client.DeleteNode(deleteNode, logChan)
	return nil
}
