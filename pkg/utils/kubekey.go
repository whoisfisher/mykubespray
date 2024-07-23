package utils

import (
	"bytes"
	"fmt"
	"github.com/offline-kubespray/pkg/entity"
	"log"
	"strings"
	"text/template"
)

type KubekeyClient struct {
	KubekeyConf entity.KubekeyConf
	OSClient    OSClient
}

func NewKubekeyClient(kubekeyConf entity.KubekeyConf, osClient OSClient) *KubekeyClient {
	return &KubekeyClient{
		KubekeyConf: kubekeyConf,
		OSClient:    osClient,
	}
}

func (client *KubekeyClient) ParseToTemplate() *entity.KubekeyTemplate {
	template := &entity.KubekeyTemplate{}
	for _, host := range client.KubekeyConf.Hosts {
		template.HostList += fmt.Sprintf("- {name: %s, address: %s, internalAddress: %s, port: %s, user: %s, password: %s}\n  ", host.Name, host.Address, host.InternalAddress, host.Port, host.User, host.Password)
	}
	template.HostList = strings.TrimSpace(template.HostList)

	for _, cp := range client.KubekeyConf.Etcds {
		template.EtcdList += fmt.Sprintf("- %s\n    ", cp)
	}
	template.EtcdList = strings.TrimSpace(template.EtcdList)

	for _, cp := range client.KubekeyConf.ContronPlanes {
		template.ControlPlaneList += fmt.Sprintf("- %s\n    ", cp)
	}
	template.ControlPlaneList = strings.TrimSpace(template.ControlPlaneList)

	for _, cp := range client.KubekeyConf.Workers {
		template.WorkerList += fmt.Sprintf("- %s\n    ", cp)
	}
	template.WorkerList = strings.TrimSpace(template.WorkerList)

	for _, ns := range client.KubekeyConf.NtpServers {
		template.NtpServerList += fmt.Sprintf("- %s\n      ", ns)
	}
	template.NtpServerList = strings.TrimSpace(template.NtpServerList)
	template.Registry += fmt.Sprintf("- %s", client.KubekeyConf.Registry.NodeName)
	template.RegistryType = client.KubekeyConf.RegistryType
	template.RegistryUrI = client.KubekeyConf.RegistryUrI
	template.RegistryUser = client.KubekeyConf.RegistryUser
	template.RegistryPassword = client.KubekeyConf.RegistryPassword
	template.ProxyMode = client.KubekeyConf.ProxyMode
	template.ContainerManager = client.KubekeyConf.ContainerManager
	template.ClusterName = client.KubekeyConf.ClusterName
	template.KubernetesVersion = client.KubekeyConf.KubernetesVersion
	template.KubeServiceCIDR = client.KubekeyConf.KubeServiceCIDR
	template.KubePodsCIDR = client.KubekeyConf.KubePodsCIDR
	template.KKPath = client.KubekeyConf.KKPath
	template.TaichuPackagePath = client.KubekeyConf.TaichuPackagePath
	template.VIPServer = client.KubekeyConf.VIPServer
	return template
}

func (client *KubekeyClient) GenerateConfig() error {
	dirPath := fmt.Sprintf("/tmp/%s", client.KubekeyConf.ClusterName)
	path := fmt.Sprintf("/tmp/%s/config-sample.yaml", client.KubekeyConf.ClusterName)
	templateText := `
apiVersion: kubekey.kubesphere.io/v1alpha2
kind: Cluster
metadata:
  name: {{ .ClusterName }}
spec:
  hosts:
  {{ .HostList }}
  roleGroups:
    etcd:
    {{ .EtcdList }}
    control-plane: 
    {{ .ControlPlaneList }}
    worker:
    {{ .WorkerList }}
    registry:
    {{ .Registry }}
  controlPlaneEndpoint:
    internalLoadbalancer: haproxy
    domain: lb.kubesphere.local
    address: ""
    port: 6443
  system:
    ntpServers:
      {{ .NtpServerList }}
    timezone: "Asia/Shanghai"
  kubernetes:
    version: {{ .KubernetesVersion }}
    clusterName: cluster.local
    autoRenewCerts: true
    containerManager: {{ .ContainerManager }}
    apiserverCertExtraSans:  
      - lb.rdev.local
    proxyMode: {{ .ProxyMode }}
  etcd:
    type: kubekey
  network:
    plugin: calico
    kubePodsCIDR: {{ .KubePodsCIDR }}
    kubeServiceCIDR: {{ .KubeServiceCIDR }}
    multusCNI:
      enabled: false
  registry:
    auths:
      "{{ .RegistryUrI }}":
        username: {{ .RegistryUser }}
        password: {{ .RegistryPassword }}
        skipTLSVerify: true
        plainHTTP: true
    privateRegistry: "{{ .RegistryUrI }}"
    namespaceOverride: "kubesphereio"
    registryMirrors: []
    insecureRegistries: []
  addons: []
`
	kubekeyTemplate := client.ParseToTemplate()
	tmpl, err := template.New("config-sample.yaml").Parse(templateText)
	if err != nil {
		log.Printf("Failed to generate template object: %s", err.Error())
		return err
	}
	var rendered bytes.Buffer
	err = tmpl.Execute(&rendered, kubekeyTemplate)
	if err != nil {
		log.Printf("Failed to generate template: %s", err.Error())
		return err
	}
	err = client.OSClient.SSExecutor.MkDirALL(dirPath, func(s string) {
		log.Println(s)
	})
	if err != nil {
		log.Printf("Failed to generate dir %s: %s", dirPath, err.Error())
		return err
	}
	command := fmt.Sprintf("echo '%s' > %s", rendered.String(), path)
	err = client.OSClient.SSExecutor.ExecuteCommandWithoutReturn(command)
	if err != nil {
		log.Printf("Failed to generate kubekey config: %s", err.Error())
		return err
	}
	return nil
}

func (client *KubekeyClient) CreateCluster(logChan chan LogEntry) error {
	command := fmt.Sprintf("%s create cluster -f /tmp/%s/config-sample.yaml -a %s --with-packages --yes", client.KubekeyConf.KKPath, client.KubekeyConf.ClusterName, client.KubekeyConf.TaichuPackagePath)
	err := client.OSClient.SSExecutor.ExecuteCommand(command, logChan)
	if err != nil {
		log.Printf("Failed to create cluster %s: %s", client.KubekeyConf.ClusterName, err.Error())
		return err
	}
	return nil
}

func (client *KubekeyClient) DeleteCluster(logChan chan LogEntry) error {
	command := fmt.Sprintf("%s delete cluster -f /tmp/$%s/config-sample.yaml --force", client.KubekeyConf.KKPath, client.KubekeyConf.ClusterName)
	err := client.OSClient.SSExecutor.ExecuteCommand(command, logChan)
	if err != nil {
		log.Printf("Failed to delete cluster %s: %s", client.KubekeyConf.ClusterName, err.Error())
		return err
	}
	return nil
}

func (client *KubekeyClient) AddNode(logChan chan LogEntry) error {
	command := fmt.Sprintf("%s add node %s-f /tmp/$%s/config-sample.yaml", client.KubekeyConf.KKPath, client.KubekeyConf.ClusterName)
	err := client.OSClient.SSExecutor.ExecuteCommand(command, logChan)
	if err != nil {
		log.Printf("Failed to add node to cluster %s: %s", client.KubekeyConf.ClusterName, err.Error())
		return err
	}
	return nil
}

func (client *KubekeyClient) DeleteNode(nodeName string, logChan chan LogEntry) error {
	command := fmt.Sprintf("%s delete node %s -f /tmp/$%s/config-sample.yaml", client.KubekeyConf.KKPath, nodeName, client.KubekeyConf.ClusterName)
	err := client.OSClient.SSExecutor.ExecuteCommand(command, logChan)
	if err != nil {
		log.Printf("Failed to delete node %s from cluster %s: %s", nodeName, client.KubekeyConf.ClusterName, err.Error())
		return err
	}
	return nil
}
