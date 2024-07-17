package pkg

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

type KubekeyConf struct {
	Hosts            []Host
	HostList         string
	EtcdList         string
	ControlPlaneList string
	WorkerList       string
	Registry         string
	VIPServer        string
	KubePodsCIDR     string
	KubeServiceCIDR  string
}

type Host struct {
	Name            string
	Address         string
	InternalAddress string
	User            string
	Password        string
	Port            int
}

type KubekeyClient struct {
	KubekeyConf KubekeyConf
	OSClient    OSClient
}

func NewKubekeyClient(kubekeyConf KubekeyConf, osClient OSClient) *KubekeyClient {
	return &KubekeyClient{
		KubekeyConf: kubekeyConf,
		OSClient:    osClient,
	}
}

func (client *KubekeyClient) GenerateConfig() error {
	templateText := `
apiVersion: kubekey.kubesphere.io/v1alpha2
kind: Cluster
metadata:
  name: sample
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
      - node1
  kubernetes:
    version: v1.24.9
    clusterName: cluster.local
    autoRenewCerts: true
    containerManager: containerd
  etcd:
    type: kubekey
  network:
    plugin: calico
    kubePodsCIDR: 10.233.64.0/18
    kubeServiceCIDR: 10.233.0.0/18
    multusCNI:
      enabled: false
  registry:
    type: harbor
    auths:
      "dockerhub.kubekey.local":
        username: admin
        password: Def@u2tpwd
        skipTLSVerify: true
        plainHTTP: true
    privateRegistry: "dockerhub.kubekey.local"
    namespaceOverride: "kubesphereio"
    registryMirrors: []
    insecureRegistries: []
  addons: []
`
	for _, host := range client.KubekeyConf.Hosts {
		client.KubekeyConf.HostList += fmt.Sprintf("- {name: %s, address: %s, internalAddress: %s, port: %s, user: %s, password: \"%s\"}    ", host.Name, host.Address, host.InternalAddress, host.Port, host.User, host.Password)
	}
	client.KubekeyConf.HostList = strings.TrimSpace(client.KubekeyConf.HostList)

	tmpl, err := template.New("keepalived.conf").Parse(templateText)
	if err != nil {
		return err
	}
	var rendered bytes.Buffer
	err = tmpl.Execute(&rendered, client.KubekeyConf)
	if err != nil {
		return err
	}
	command := fmt.Sprintf("echo '%s' > %s", rendered.String(), configFile)
	err = client.OSClient.SSExecutor.ExecuteCommandWithoutReturn(command)
	if err != nil {
		return err
	}
	return nil
}
