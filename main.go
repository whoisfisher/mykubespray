package main

import (
	"fmt"
	"github.com/toolkits/pkg/runner"
	"github.com/urfave/cli/v2"
	"github.com/whoisfisher/mykubespray/pkg/entity"
	"github.com/whoisfisher/mykubespray/pkg/server"
	"github.com/whoisfisher/mykubespray/pkg/utils/etcd"
	"github.com/whoisfisher/mykubespray/pkg/utils/oss"
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
				Usage:   "Specify configuration file(.toml)",
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

func main1() {
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

func main2() {
	host := entity.Host{
		Name:            "kylin2",
		Address:         "192.168.227.149",
		InternalAddress: "192.168.227.149",
		Port:            22,
		User:            "root",
		Password:        "Def@u1tpwd",
	}
	s3Client, err := oss.NewS3("172.30.1.12:30204", "admin", "Def@u1tpwd", "etcd", "us-east-1", false)
	if err != nil {
		panic(err)
	}
	bm := etcd.NewBackupManager(host, "/data", "c:/tmp", "wangzhendong", s3Client)
	bm.BackupEtcd()
}

func main() {
	hosts := []entity.Host{
		{
			Name:            "kylin2",
			Address:         "192.168.227.149",
			InternalAddress: "192.168.227.149",
			Port:            22,
			User:            "root",
			Password:        "Def@u1tpwd",
		},
	}
	s3Client, err := oss.NewS3("172.30.1.12:30204", "admin", "Def@u1tpwd", "etcd", "us-east-1", false)
	if err != nil {
		panic(err)
	}

	err = etcd.RestoreEtcdCluster(hosts, "/data", "c:/tmp", "wangzhendong", "etcd-backup-1732083472.db", s3Client)
	if err != nil {
		return
	}
}
