package utils

import (
	"fmt"
	"github.com/whoisfisher/mykubespray/pkg/entity"
	"github.com/whoisfisher/mykubespray/pkg/logger"
	"golang.org/x/crypto/ssh"
	"os"
)

type SSHConfig struct {
	Host        string
	Port        int32
	User        string
	PrivateKey  string
	Password    string
	AuthMethods []ssh.AuthMethod
}

// SSHConnection manages an SSH connection.
type SSHConnection struct {
	Client *ssh.Client
}

func NewConnection(host entity.Host) (*SSHConnection, error) {
	sshConfig := &ssh.ClientConfig{
		User:            host.User,
		Auth:            host.AuthMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if host.Password != "" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(host.Password))
	}

	if host.PrivateKey != "" {
		key, err := parsePrivateKey(host.PrivateKey)
		if err != nil {
			logger.GetLogger().Errorf("Failed to parse private key: %s", err.Error())
			return nil, err
		}
		sshConfig.Auth = append(sshConfig.Auth, key)
	}

	address := fmt.Sprintf("%s:%d", host.Address, host.Port)
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		logger.GetLogger().Errorf("Failed to dial: %s", err.Error())
		return nil, err
	}

	connection := &SSHConnection{
		Client: client,
	}

	return connection, nil
}

// NewSSHConnection establishes a new SSH connection.
func NewSSHConnection(config SSHConfig) (*SSHConnection, error) {
	sshConfig := &ssh.ClientConfig{
		User:            config.User,
		Auth:            config.AuthMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if config.Password != "" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(config.Password))
	}

	if config.PrivateKey != "" {
		key, err := parsePrivateKey(config.PrivateKey)
		if err != nil {
			logger.GetLogger().Errorf("Failed to parse private key: %s", err.Error())
			return nil, err
		}
		sshConfig.Auth = append(sshConfig.Auth, key)
	}

	address := fmt.Sprintf("%s:%d", config.Host, config.Port)
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		logger.GetLogger().Errorf("Failed to dial: %s", err.Error())
		return nil, err
	}

	connection := &SSHConnection{
		Client: client,
	}

	return connection, nil
}

func parsePrivateKey(keyPath string) (ssh.AuthMethod, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		logger.GetLogger().Errorf("Failed to read private key file: %s", err.Error())
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		logger.GetLogger().Errorf("Failed to parse private key: %s", err.Error())
		return nil, err
	}

	return ssh.PublicKeys(signer), nil
}

// Connect establishes an SSH connection.
func (conn *SSHConnection) Connect(config SSHConfig) error {
	// No additional implementation needed, as NewSSHConnection already establishes the connection.
	return nil
}
