package pkg

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
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
			return nil, fmt.Errorf("failed to parse private key: %s", err)
		}
		sshConfig.Auth = append(sshConfig.Auth, key)
	}

	address := fmt.Sprintf("%s:%d", config.Host, config.Port)
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %s", err)
	}

	connection := &SSHConnection{
		Client: client,
	}

	return connection, nil
}

func parsePrivateKey(keyPath string) (ssh.AuthMethod, error) {
	key, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %s", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %s", err)
	}

	return ssh.PublicKeys(signer), nil
}

// Connect establishes an SSH connection.
func (conn *SSHConnection) Connect(config SSHConfig) error {
	// No additional implementation needed, as NewSSHConnection already establishes the connection.
	return nil
}
