package utils

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"log"
	"os"
)

type SSHConfig struct {
	Host        string
	Port        int
	User        string
	PrivateKey  string
	Password    string
	AuthMethods []ssh.AuthMethod
}

// SSHConnection manages an SSH connection.
type SSHConnection struct {
	Client *ssh.Client
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
			log.Printf("Failed to parse private key: %s", err.Error())
			return nil, err
		}
		sshConfig.Auth = append(sshConfig.Auth, key)
	}

	address := fmt.Sprintf("%s:%d", config.Host, config.Port)
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		log.Printf("Failed to dial: %s", err.Error())
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
		log.Printf("Failed to read private key file: %s", err.Error())
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Printf("Failed to parse private key: %s", err.Error())
		return nil, err
	}

	return ssh.PublicKeys(signer), nil
}

// Connect establishes an SSH connection.
func (conn *SSHConnection) Connect(config SSHConfig) error {
	// No additional implementation needed, as NewSSHConnection already establishes the connection.
	return nil
}
