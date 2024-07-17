package main

import (
	"fmt"
	"github.com/offline-kubespray/pkg"
	"log"
	"os"
)

func main() {
	// Example usage with SSHExecutor
	sshConfig := pkg.SSHConfig{
		Host:     "172.30.1.12",
		Port:     22,
		User:     "root",
		Password: "Def@u1tpwd",
	}
	logChan := make(chan pkg.LogEntry)
	go func() {
		for logEntry := range logChan {
			if logEntry.IsError {
				fmt.Fprintf(os.Stderr, "[ERROR] %s\n", logEntry.Message)
			} else {
				fmt.Printf("[INFO] %s\n", logEntry.Message)
			}
		}
	}()

	connection, err := pkg.NewSSHConnection(sshConfig)

	if err != nil {
		log.Fatalf("Failed to create SSH connection: %s", err)
	}

	osCOnf := pkg.OSConf{}
	localExecutor := pkg.NewLocalExecutor()
	sshExecutor := pkg.NewSSHExecutor(*connection)
	osclient := pkg.NewOSClient(osCOnf, *sshExecutor, *localExecutor)

	haproxyConf := pkg.HaproxyConf{
		Servers: []string{"112.168.32.32:6443", "112.168.31.33:6443", "112.168.31.33:6443"},
	}

	client := pkg.NewHaproxyClient(haproxyConf, *osclient)
	fmt.Println(client.OSClient.GetSpecifyNetCard(sshConfig.Host))
	fmt.Println(client.OSClient.OSConf.NetCardList)

	err = client.InstallHaproxy(sshConfig, logChan)
	if err != nil {
		log.Fatalf("Failed to execute command: %s", err)
	}
	err = client.ConfigureHaproxy()
	if err != nil {
		log.Fatalf("Failed to execute command: %s", err)
	}
	client.OSClient.DaemonReload()
	client.OSClient.StartService("haproxy")
}

//func main() {
//	// Example usage with SSHExecutor
//	sshConfig := pkg.SSHConfig{
//		Host:     "172.30.1.12",
//		Port:     22,
//		User:     "root",
//		Password: "Def@u1tpwd",
//	}
//	logChan := make(chan pkg.LogEntry)
//	go func() {
//		for logEntry := range logChan {
//			if logEntry.IsError {
//				fmt.Fprintf(os.Stderr, "[ERROR] %s\n", logEntry.Message)
//			} else {
//				fmt.Printf("[INFO] %s\n", logEntry.Message)
//			}
//		}
//	}()
//
//	connection, err := pkg.NewSSHConnection(sshConfig)
//
//	if err != nil {
//		log.Fatalf("Failed to create SSH connection: %s", err)
//	}
//
//	osCOnf := pkg.OSConf{}
//	localExecutor := pkg.NewLocalExecutor()
//	sshExecutor := pkg.NewSSHExecutor(*connection)
//	osclient := pkg.NewOSClient(osCOnf, *sshExecutor, *localExecutor)
//
//	keepalivedConf := pkg.KeepalivedConf{
//		State:    "MASTER",
//		IntFace:  "enp4s0",
//		Priority: 100,
//		AuthType: "PASS",
//		AuthPass: "2222",
//		SrcIP:    "182.168.31.31",
//		Peers:    []string{"182.168.32.32", "182.168.31.33"},
//		VIP:      "182.168.21.21",
//	}
//
//	client := pkg.NewKeepAlivedClient(keepalivedConf, *osclient)
//	fmt.Println(client.OSClient.GetSpecifyNetCard(sshConfig.Host))
//	fmt.Println(client.OSClient.OSConf.NetCardList)
//
//	err = client.InstallKeepalived(sshConfig, logChan)
//	if err != nil {
//		log.Fatalf("Failed to execute command: %s", err)
//	}
//	err = client.ConfigureKeepalived()
//	if err != nil {
//		log.Fatalf("Failed to execute command: %s", err)
//	}
//	client.OSClient.DaemonReload()
//	client.OSClient.StartService("keepalived")
//}

//func main() {
//	// Example usage with SSHExecutor
//	sshConfig := pkg.SSHConfig{
//		Host:     "172.30.1.12",
//		Port:     22,
//		User:     "root",
//		Password: "Def@u1tpwd",
//	}
//	connection, err := pkg.NewSSHConnection(sshConfig)
//	if err != nil {
//		log.Fatalf("Failed to create SSH connection: %s", err)
//	}
//
//	sshExecutor := pkg.NewSSHExecutor(*connection)
//
//	logChan := make(chan pkg.LogEntry)
//	go func() {
//		for logEntry := range logChan {
//			if logEntry.IsError {
//				fmt.Fprintf(os.Stderr, "[ERROR] %s\n", logEntry.Message)
//			} else {
//				fmt.Printf("[INFO] %s\n", logEntry.Message)
//			}
//		}
//	}()
//
//	err = sshExecutor.ExecuteCommand("ping 127.0.0.1", logChan)
//	if err != nil {
//		log.Fatalf("Failed to execute command: %s", err)
//	}
//}

//func main() {
//	localExecutor := pkg.NewLocalExecutor()
//	logChan := make(chan pkg.LogEntry)
//	go func() {
//		for logEntry := range logChan {
//			if logEntry.IsError {
//				fmt.Fprintf(os.Stderr, "[ERROR] %s\n", logEntry.Message)
//			} else {
//				fmt.Printf("[INFO] %s\n", logEntry.Message)
//			}
//		}
//	}()
//	err := localExecutor.ExecuteCommand("ping 127.0.0.1 -t", logChan)
//	if err != nil {
//		log.Fatalf("Failed to execute command: %s", err)
//	}
//}
