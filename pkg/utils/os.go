package utils

import (
	"fmt"
	"github.com/whoisfisher/mykubespray/pkg/logger"
	"log"
	"strings"
)

type OSConf struct {
	Arch           string
	Version        string
	Name           string
	CPU            string
	CPUCores       string
	MemorySize     string
	DiskSize       string
	NetCardList    []string
	SpecifyNetCard string
}

type OSClient struct {
	OSConf        OSConf
	SSExecutor    SSHExecutor
	LocalExecutor LocalExecutor
}

func NewOSClient(osConf OSConf, sshExecutor SSHExecutor, localExecutor LocalExecutor) *OSClient {
	osclient := &OSClient{
		OSConf:        osConf,
		SSExecutor:    sshExecutor,
		LocalExecutor: localExecutor,
	}
	osclient.GetOSConf()
	osclient.GetCPU()
	osclient.GetCPUCores()
	osclient.GetMemorySize()
	osclient.GetDiskSize()
	osclient.GetNetCardList()
	return osclient
}

func (client *OSClient) GetOSConf() bool {
	output, err := client.SSExecutor.ExecuteShortCommand("cat /etc/os-release")
	if err != nil {
		return false
	}
	client.OSConf.Name = "Unknown"
	client.OSConf.Version = "Unknown"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ID=") {
			client.OSConf.Name = strings.TrimPrefix(line, "ID=")
		} else if strings.HasPrefix(line, "VERSION_ID=") {
			client.OSConf.Version = strings.TrimPrefix(line, "VERSION_ID=")
		}
	}
	res, err := client.SSExecutor.ExecuteShortCommand("arch")
	if err != nil {
		log.Printf("Failed to get os arch: %s", err.Error())
		client.OSConf.Arch = "Unknown"
		return false
	}
	arch := strings.TrimSpace(string(res))
	client.OSConf.Arch = arch
	return true
}

func (client *OSClient) GetDistribution() (string, error) {
	output, err := client.SSExecutor.ExecuteShortCommand("cat /etc/os-release")
	if err != nil {
		logger.GetLogger().Errorf("Failed to get distribution: %s", err.Error())
		return "", err
	}
	res := parseOSRelease(output)
	return res, nil
}

func (client *OSClient) DaemonReload() error {
	_, err := client.SSExecutor.ExecuteShortCommand("systemctl daemon-reload")
	if err != nil {
		logger.GetLogger().Errorf("Failed to reload daemon: %s", err.Error())
		return err
	}
	return nil
}

func (client *OSClient) RestartService(service string) error {
	command := fmt.Sprintf("systemctl restart %s", service)
	_, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		logger.GetLogger().Errorf("Failed to restart %s: %s", service, err.Error())
		return err
	}
	return nil
}

func (client *OSClient) StartService(service string) error {
	command := fmt.Sprintf("systemctl start %s", service)
	_, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		logger.GetLogger().Errorf("Failed to start %s: %s", service, err.Error())
		return err
	}
	return nil
}

func (client *OSClient) StopService(service string) error {
	command := fmt.Sprintf("systemctl stop %s", service)
	_, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		log.Printf("Failed to stop %s: %s", service, err.Error())
		return err
	}
	return nil
}

func (client *OSClient) DisableService(service string) error {
	command := fmt.Sprintf("systemctl disable %s", service)
	_, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		log.Printf("Failed to disable %s: %s", service, err.Error())
		return err
	}
	return nil
}

func (client *OSClient) EnableService(service string) error {
	command := fmt.Sprintf("systemctl enable %s", service)
	_, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		log.Printf("Failed to enable %s: %s", service, err.Error())
		return err
	}
	return nil
}

func (client *OSClient) MaskService(service string) error {
	command := fmt.Sprintf("systemctl mask %s", service)
	_, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		log.Printf("Failed to mask %s: %s", service, err.Error())
		return err
	}
	return nil
}

func (client *OSClient) UNMaskService(service string) error {
	command := fmt.Sprintf("systemctl unmask %s", service)
	_, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		log.Printf("Failed to unmask %s: %s", service, err.Error())
		return err
	}
	return nil
}

func (client *OSClient) StatusService(service string) bool {
	command := fmt.Sprintf("systemctl status %s | grep -iE active", service)
	res, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		log.Printf("Failed to view %s status: %s", service, err.Error())
		return false
	}
	if strings.Contains(res, "inactive") {
		return false
	}
	return true
}

func (client *OSClient) GetCPUCores() bool {
	command := "grep -c ^processor /proc/cpuinfo"
	res, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		log.Printf("Failed to get cpu cores: %s", err.Error())
		client.OSConf.CPUCores = "Unknown"
		return false
	}
	cores := strings.TrimSpace(res)
	client.OSConf.CPUCores = cores
	return true
}

func (client *OSClient) GetCPU() bool {
	command := "grep -iE \"^model\\s+name\\s+:\" /proc/cpuinfo | awk -F':' '{print $NF}' | sort -u"
	res, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		log.Printf("Failed to get cpu info: %s", err.Error())
		client.OSConf.CPU = "Unknown"
		return false
	}
	cpu := strings.TrimSpace(res)
	client.OSConf.CPU = cpu
	return true
}

func (client *OSClient) GetMemorySize() bool {
	command := "free -m | grep Mem | awk '{print $2}'"
	res, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		log.Printf("Failed to get memory info: %s", err.Error())
		client.OSConf.MemorySize = "Unknown"
		return false
	}
	memCapacity := strings.TrimSpace(res) + "MB"
	client.OSConf.MemorySize = memCapacity
	return true
}

func (client *OSClient) GetDiskSize() bool {
	command := "df -h / | tail -n 1 | awk '{print $2}'"
	res, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		log.Printf("Failed to get disk size: %s", err.Error())
		client.OSConf.DiskSize = "Unknown"
		return false
	}
	diskCapacity := strings.TrimSpace(res)
	client.OSConf.DiskSize = diskCapacity
	return true
}

func (client *OSClient) GetNetCardList() bool {
	command := "ip addr show | grep -o '^[0-9]\\+: [a-zA-Z0-9]*' | awk '{print $2}'"
	res, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		log.Printf("Failed to get netcard list: %s", err.Error())
		client.OSConf.NetCardList = []string{"Unknown"}
		return false
	}
	client.OSConf.NetCardList = strings.Split(res, " ")
	return true
}

func (client *OSClient) GetSpecifyNetCard(ipaddr string) string {
	command := fmt.Sprintf("ip addr | grep -B 2 '%s' | head -n 1 | awk -F':' '{print $2}'", ipaddr)
	res, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		log.Printf("Failed to get netcard info for %s: %s", ipaddr, err.Error())
		client.OSConf.SpecifyNetCard = res
	}
	client.OSConf.SpecifyNetCard = strings.TrimSpace(res)
	return client.OSConf.SpecifyNetCard
}

func (client *OSClient) IsProcessExist(processName string) bool {
	
}
