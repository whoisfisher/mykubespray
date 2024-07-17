package pkg

import (
	"fmt"
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
		return "", err
	}
	res := parseOSRelease(output)
	return res, nil
}

func (client *OSClient) DaemonReload() bool {
	_, err := client.SSExecutor.ExecuteShortCommand("systemctl daemon-reload")
	if err != nil {
		return false
	}
	return true
}

func (client *OSClient) RestartService(service string) bool {
	command := fmt.Sprintf("systemctl restart %s", service)
	_, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		return false
	}
	return true
}

func (client *OSClient) StartService(service string) bool {
	command := fmt.Sprintf("systemctl start %s", service)
	_, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		return false
	}
	return true
}

func (client *OSClient) StopService(service string) bool {
	command := fmt.Sprintf("systemctl stop %s", service)
	_, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		return false
	}
	return true
}

func (client *OSClient) DisableService(service string) bool {
	command := fmt.Sprintf("systemctl disable %s", service)
	_, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		return false
	}
	return true
}

func (client *OSClient) EnableService(service string) bool {
	command := fmt.Sprintf("systemctl enable %s", service)
	_, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		return false
	}
	return true
}

func (client *OSClient) MaskService(service string) bool {
	command := fmt.Sprintf("systemctl mask %s", service)
	_, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		return false
	}
	return true
}

func (client *OSClient) UNMaskService(service string) bool {
	command := fmt.Sprintf("systemctl unmask %s", service)
	_, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		return false
	}
	return true
}

func (client *OSClient) StatusService(service string) bool {
	command := fmt.Sprintf("systemctl status %s | grep -iE active", service)
	res, err := client.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
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
		client.OSConf.SpecifyNetCard = res
	}
	client.OSConf.SpecifyNetCard = strings.TrimSpace(res)
	return client.OSConf.SpecifyNetCard
}
