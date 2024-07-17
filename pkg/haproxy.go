package pkg

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"text/template"
)

type HaproxyConf struct {
	Servers    []string
	StrServers string
}

type HaproxyClient struct {
	HaproxyConf HaproxyConf
	OSClient    OSClient
}

func NewHaproxyClient(haproxyConf HaproxyConf, osClient OSClient) *HaproxyClient {
	return &HaproxyClient{
		HaproxyConf: haproxyConf,
		OSClient:    osClient,
	}
}

func (client *HaproxyClient) InstallHaproxy(sshConfig SSHConfig, logChan chan LogEntry) error {
	command := ""
	os, err := GetDistribution(sshConfig)
	if err != nil {
		log.Fatalf("Failed to create SSH connection: %s", err)
		return err
	}
	if os == "ubuntu" {
		command = "sudo apt install haproxy -y"
	} else if os == "centos" {
		command = "sudo yum install haproxy -y"
	}
	err = client.OSClient.SSExecutor.ExecuteCommand(command, logChan)
	if err != nil {
		log.Fatalf("Failed to execute command: %s", err)
		return err
	}
	return nil
}

func (client *HaproxyClient) ConfigureHaproxy() error {
	configFile := "/etc/haproxy/haproxy.cfg"
	templateText := `
global
  log /dev/log  local0 warning
  chroot      /var/lib/haproxy
  pidfile     /var/run/haproxy.pid
  maxconn     4000
  user        haproxy
  group       haproxy
  daemon
  stats socket /var/lib/haproxy/stats
defaults
  log global
  option  httplog
  option  dontlognull
  timeout connect 5000
  timeout client 50000
  timeout server 50000
frontend kube-apiserver
  bind *:6443
  mode tcp
  option tcplog
  default_backend kube-apiserver
backend kube-apiserver
  mode tcp
  option tcplog
  option tcp-check
  balance roundrobin
  default-server inter 10s downinter 5s rise 2 fall 2 slowstart 60s maxconn 250 maxqueue 256 weight 100
  {{ .StrServers }}
	`
	for index, server := range client.HaproxyConf.Servers {
		client.HaproxyConf.StrServers += fmt.Sprintf("server kube-apiserver-%d %s check\n  ", index, server)
	}
	client.HaproxyConf.StrServers = strings.TrimSpace(client.HaproxyConf.StrServers)
	tmpl, err := template.New("haproxy.conf").Parse(templateText)
	if err != nil {
		return err
	}
	var rendered bytes.Buffer
	err = tmpl.Execute(&rendered, client.HaproxyConf)
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
