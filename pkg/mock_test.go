package pkg

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
	"testing"
)

// MockSSHConnection 实现了 SSHConnection 接口，用于测试
type MockSSHConnection struct{}

type MockSSHClient struct{}

func (c *MockSSHConnection) ConnectSSH(config SSHConfig) (SSHConnection, error) {
	//sshConfig := &ssh.ClientConfig{
	//	User:            config.User,
	//	Auth:            config.AuthMethods,
	//	HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	//}
	//
	//address := fmt.Sprintf("%s:%d", config.Host, config.Port)
	//client, err := ssh.Dial("tcp", address, sshConfig)
	//if err != nil {
	//	return SSHConnection{}, fmt.Errorf("Failed to dial: %s", err)
	//}
	mockClient := &MockSSHClient{}
	connection := SSHConnection{
		Client: mockClient,
	}

	return connection, nil
}

func TestSSHExecutor_ExecuteCommand(t *testing.T) {
	mockSSH := &MockSSHConnection{}
	connection, err := mockSSH.ConnectSSH(SSHConfig{
		User: "username",
		AuthMethods: []ssh.AuthMethod{
			ssh.Password("password"),
		},
		Host: "hostname",
		Port: 22,
	})
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	sshExecutor := NewSSHExecutor(connection)
	logChan := make(chan LogEntry)

	// 测试执行命令是否能够成功
	err = sshExecutor.ExecuteCommand("ls -l", logChan)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	// 测试执行错误命令是否能够捕获错误
	err = sshExecutor.ExecuteCommand("non_existing_command", logChan)
	if err == nil {
		t.Error("Expected an error, but got none")
	}
}

func TestSSHExecutor_CopyFile(t *testing.T) {
	mockSSH := &MockSSHConnection{}
	connection, err := mockSSH.ConnectSSH(SSHConfig{
		User: "username",
		AuthMethods: []ssh.AuthMethod{
			ssh.Password("password"),
		},
		Host: "hostname",
		Port: 22,
	})
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	sshExecutor := NewSSHExecutor(connection)
	outputHandler := func(msg string) {
		fmt.Println(msg)
	}

	srcFile := "testdata/source.txt"
	destFile := "testdata/destination.txt"

	// 创建测试用的源文件
	src, err := os.Create(srcFile)
	if err != nil {
		t.Fatalf("Failed to create source file: %s", err)
	}
	defer src.Close()
	src.WriteString("Hello, world!")
	src.Close()

	// 测试复制文件是否能够成功
	err = sshExecutor.CopyFile(srcFile, destFile, outputHandler)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	logChan := make(chan LogEntry)
	// 清理测试用的目标文件
	err = sshExecutor.ExecuteCommand(fmt.Sprintf("rm %s", destFile), logChan)
	if err != nil {
		t.Fatalf("Failed to remove destination file: %s", err)
	}
}
