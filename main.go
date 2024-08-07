package main

import (
	"fmt"
	"github.com/toolkits/pkg/runner"
	"github.com/urfave/cli/v2"
	"github.com/whoisfisher/mykubespray/pkg/entity"
	"github.com/whoisfisher/mykubespray/pkg/server"
	"github.com/whoisfisher/mykubespray/pkg/utils"
	"log"
	"os"
)

var VERSION = "not specified"

func printEnv() {
	runner.Init()
	fmt.Println("runner.cwd:", runner.Cwd)
	fmt.Println("runner.hostname:", runner.Hostname)
	fmt.Println("runner.fd_limits:", runner.FdLimits())
	fmt.Println("runner.vm_limits:", runner.VMLimits())
}

func NewServerCmd() *cli.Command {
	return &cli.Command{
		Name:  "server",
		Usage: "Run Server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "conf",
				Aliases: []string{"c"},
				Usage:   "Spefify configuration file(.toml)",
			},
		},
		Action: func(context *cli.Context) error {
			printEnv()
			var options []server.ServerOption
			if context.String("conf") != "" {
				options = append(options, server.SetConfigFile(context.String("conf")))
			}
			options = append(options, server.SetVersion(VERSION))
			server.Run(options...)
			return nil
		},
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "cluster-utils"
	app.Version = "1.0.0"
	app.Usage = "cluster-utils"
	app.Commands = []*cli.Command{
		NewServerCmd(),
	}
	err := app.Run(os.Args)
	if err != nil {
		return
	}
}

//func main() {
//	// Example usage with SSHExecutor
//	test2_install_cluster()
//}

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
//	haproxyConf := pkg.HaproxyConf{
//		Servers: []string{"112.168.32.32:6443", "112.168.31.33:6443", "112.168.31.33:6443"},
//	}
//
//	client := pkg.NewHaproxyClient(haproxyConf, *osclient)
//	fmt.Println(client.OSClient.GetSpecifyNetCard(sshConfig.Host))
//	fmt.Println(client.OSClient.OSConf.NetCardList)
//
//	err = client.InstallHaproxy(sshConfig, logChan)
//	if err != nil {
//		log.Fatalf("Failed to execute command: %s", err)
//	}
//	err = client.ConfigureHaproxy()
//	if err != nil {
//		log.Fatalf("Failed to execute command: %s", err)
//	}
//	client.OSClient.DaemonReload()
//	client.OSClient.StartService("haproxy")
//}

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
//	HaproxyConf := pkg.HaproxyConf{
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
//	client := pkg.NewHaproxyClient(HaproxyConf, *osclient)
//	fmt.Println(client.OSClient.GetSpecifyNetCard(sshConfig.Host))
//	fmt.Println(client.OSClient.OSConf.NetCardList)
//
//	err = client.InstallHaproxy(sshConfig, logChan)
//	if err != nil {
//		log.Fatalf("Failed to execute command: %s", err)
//	}
//	err = client.ConfigureHaproxy()
//	if err != nil {
//		log.Fatalf("Failed to execute command: %s", err)
//	}
//	client.OSClient.DaemonReload()
//	client.OSClient.StartService("Haproxy")
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

func test_install_cluster() {
	sshConfig := utils.SSHConfig{
		Host:     "172.30.1.13",
		Port:     22,
		User:     "root",
		Password: "Def@u1tpwd",
	}
	logChan := make(chan utils.LogEntry)
	go func() {
		for logEntry := range logChan {
			if logEntry.IsError {
				fmt.Fprintf(os.Stderr, "[ERROR] %s\n", logEntry.Message)
			} else {
				fmt.Printf("[INFO] %s\n", logEntry.Message)
			}
		}
	}()

	connection, err := utils.NewSSHConnection(sshConfig)

	if err != nil {
		log.Fatalf("Failed to create SSH connection: %s", err)
	}

	osCOnf := utils.OSConf{}
	localExecutor := utils.NewLocalExecutor()
	sshExecutor := utils.NewSSHExecutor(*connection)
	osclient := utils.NewOSClient(osCOnf, *sshExecutor, *localExecutor)

	kubekeyConf := entity.KubekeyConf{
		ClusterName: "wangcluster",
		Hosts: []entity.Host{{
			Name:            "node1",
			Address:         "1.1.1.1",
			InternalAddress: "1.1.1.1",
			Port:            2222,
			User:            "root",
			Password:        "Def@u1tpwd",
		}, {
			Name:            "node2",
			Address:         "1.1.1.2",
			InternalAddress: "1.1.1.2",
			Port:            2222,
			User:            "root",
			Password:        "Def@u1tpwd",
		}, {
			Name:            "node3",
			Address:         "1.1.1.3",
			InternalAddress: "1.1.1.3",
			Port:            2222,
			User:            "root",
			Password:        "Def@u1tpwd",
		}, {
			Name:            "node4",
			Address:         "1.1.1.4",
			InternalAddress: "1.1.1.4",
			Port:            2222,
			User:            "root",
			Password:        "Def@u1tpwd",
		}, {
			Name:            "node5",
			Address:         "1.1.1.5",
			InternalAddress: "1.1.1.5",
			Port:            2222,
			User:            "root",
			Password:        "Def@u1tpwd",
		}, {
			Name:            "node6",
			Address:         "1.1.1.6",
			InternalAddress: "1.1.1.6",
			Port:            2222,
			User:            "root",
			Password:        "Def@u1tpwd",
		},
		},
		Etcds:             []string{"node1", "node2", "node3"},
		ContronPlanes:     []string{"node1", "node2", "node3"},
		Workers:           []string{"node1", "node2", "node3", "node4", "node5"},
		NtpServers:        []string{"node1", "aliyun.com"},
		KubernetesVersion: "v1.24.9",
		ContainerManager:  "containerd",
		ProxyMode:         "iptables",
		Registry: entity.Registry{
			NodeName:  "node6",
			Url:       "dockerhub.kubekey.local",
			User:      "admin",
			Password:  "Def@u1tpwd",
			SkipTLS:   false,
			PlainHttp: false,
			Type:      "harbor",
		},
		KubePodsCIDR:      "10.233.64.0/18",
		KubeServiceCIDR:   "10.233.0.0/18",
		KKPath:            "/root/cluster1/kk",
		TaichuPackagePath: "/root/cluster1/kubesphere2.tar.gz",
	}
	client := utils.NewKubekeyClient(kubekeyConf, *osclient)
	client.GenerateConfig()
	client.CreateCluster(logChan)
}

func test2_install_cluster() {
	sshConfig := utils.SSHConfig{
		Host:     "192.168.227.144",
		Port:     22,
		User:     "root",
		Password: "Def@u1tpwd",
	}
	logChan := make(chan utils.LogEntry)
	go func() {
		for logEntry := range logChan {
			if logEntry.IsError {
				fmt.Fprintf(os.Stderr, "[ERROR] %s\n", logEntry.Message)
			} else {
				fmt.Printf("[INFO] %s\n", logEntry.Message)
			}
		}
	}()

	connection, err := utils.NewSSHConnection(sshConfig)

	if err != nil {
		log.Fatalf("Failed to create SSH connection: %s", err)
	}

	osCOnf := utils.OSConf{}
	localExecutor := utils.NewLocalExecutor()
	sshExecutor := utils.NewSSHExecutor(*connection)
	osclient := utils.NewOSClient(osCOnf, *sshExecutor, *localExecutor)

	kubekeyConf := entity.KubekeyConf{
		ClusterName: "wangcluster",
		Hosts: []entity.Host{{
			Name:            "node1",
			Address:         "192.168.227.145",
			InternalAddress: "192.168.227.145",
			Port:            22,
			User:            "root",
			Password:        "Def@u1tpwd",
		}, {
			Name:            "registry",
			Address:         "192.168.227.144",
			InternalAddress: "192.168.227.144",
			Port:            22,
			User:            "root",
			Password:        "Def@u1tpwd",
		}},
		Etcds:             []string{"node1"},
		ContronPlanes:     []string{"node1"},
		Workers:           []string{"node1"},
		NtpServers:        []string{"node1", "aliyun.com"},
		KubernetesVersion: "v1.24.9",
		ContainerManager:  "containerd",
		ProxyMode:         "iptables",
		Registry: entity.Registry{
			Name:      "myharbor",
			NodeName:  "registry",
			Url:       "dockerhub.kubekey.local",
			User:      "admin",
			Password:  "Def@u1tpwd",
			SkipTLS:   false,
			PlainHttp: false,
			Type:      "harbor",
		},
		KubePodsCIDR:      "10.233.64.0/18",
		KubeServiceCIDR:   "10.233.0.0/18",
		KKPath:            "/root/cluster1/kk",
		TaichuPackagePath: "/root/cluster1/taichu2.tar.gz",
	}
	client := utils.NewKubekeyClient(kubekeyConf, *osclient)
	client.GenerateConfig()
	client.CreateCluster(logChan)
}
