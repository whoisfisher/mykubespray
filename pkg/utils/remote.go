package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/pkg/sftp"
	"github.com/whoisfisher/mykubespray/pkg/entity"
	"github.com/whoisfisher/mykubespray/pkg/logger"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// SSHExecutor implements Executor for SSH connections.
type SSHExecutor struct {
	Connection SSHConnection
}

func NewExecutor(host entity.Host) *SSHExecutor {
	connection, err := NewConnection(host)
	if err != nil {
		return nil
	}
	return &SSHExecutor{Connection: *connection}
}

// NewSSHExecutor creates a new instance of SSHExecutor.
func NewSSHExecutor(connection SSHConnection) *SSHExecutor {
	return &SSHExecutor{Connection: connection}
}

// ExecuteCommand executes a command over SSH.

func (executor *SSHExecutor) ExecuteShortCommand(command string) (string, error) {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SSH session: %s", err.Error())
		return "", err
	}
	defer session.Close()
	res, err := session.CombinedOutput(command)
	if err != nil {
		logger.GetLogger().Errorf("Failed to execute command: %s, %s", err.Error(), res)
		return "", err
	}
	return string(res), nil
}

func (executor *SSHExecutor) ExecuteShortCMD(command string) ([]byte, error) {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SSH session: %s", err.Error())
		return nil, err
	}
	defer session.Close()
	res, err := session.CombinedOutput(command)
	if err != nil {
		logger.GetLogger().Errorf("Failed to execute command: %s", err.Error())
		return nil, err
	}
	return res, nil
}

func (executor *SSHExecutor) ExecuteCommandOld(command string, logChan chan LogEntry) error {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SSH session: %s", err.Error())
		return err
	}
	defer session.Close()
	//session.RequestPty("xterm", 80, 40, ssh.TerminalModes{})

	stdin, err := session.StdinPipe()
	if err != nil {
		logger.GetLogger().Errorf("Unable to setup stdin for session: %v", err)
		return err
	}

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		logger.GetLogger().Errorf("Unable to create stdout pipe: %v", err.Error())
		return err
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create stderr pipe: %s", err.Error())
		return err
	}

	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			go fmt.Fprintln(stdin, "yes\n")
			text := scanner.Text()
			if strings.Contains(text, "[yes/no]") {
				continue
			} else {
				logChan <- LogEntry{Message: text, IsError: false}
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			go fmt.Fprintln(stdin, "yes\n")
			text := scanner.Text()
			if strings.Contains(text, "[yes/no]") {
				continue
			} else {
				logChan <- LogEntry{Message: text, IsError: false}
			}
		}
	}()

	err = session.Start(command)
	if err != nil {
		logChan <- LogEntry{Message: "Pipeline Done", IsError: true}
		logger.GetLogger().Errorf("Failed to run SSH command: %s", err.Error())
		return err
	}

	err = session.Wait()
	if err != nil {
		logger.GetLogger().Errorf("SSH command execution failed: %s", err.Error())
		logChan <- LogEntry{Message: "Pipeline Done", IsError: true}
		return err
	}
	logChan <- LogEntry{Message: "Pipeline Done", IsError: false}
	return nil
}

func (executor *SSHExecutor) ExecuteCommand(command string, logChan chan LogEntry) error {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SSH session: %s", err.Error())
		return err
	}
	defer session.Close()
	//session.RequestPty("xterm", 80, 40, ssh.TerminalModes{})

	stdin, err := session.StdinPipe()
	if err != nil {
		logger.GetLogger().Errorf("Unable to setup stdin for session: %v", err)
		return err
	}

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		logger.GetLogger().Errorf("Unable to create stdout pipe: %v", err.Error())
		return err
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create stderr pipe: %s", err.Error())
		return err
	}

	var wg sync.WaitGroup

	doneStdout := make(chan struct{})
	doneStderr := make(chan struct{})

	wg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.GetLogger().Errorf("Recovered from panic in stdout pipe: %v", r)
			}
			wg.Done()
		}()
		defer close(doneStdout)
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			go fmt.Fprintln(stdin, "yes\n")
			text := scanner.Text()
			if strings.Contains(text, "[yes/no]") {
				continue
			} else {
				select {
				case logChan <- LogEntry{Message: text, IsError: false}:
				case <-doneStdout:
					return
				}
			}
		}
		if err := scanner.Err(); err != nil {
			logger.GetLogger().Errorf("Error reading stdout pipe: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.GetLogger().Errorf("Recovered from panic in stderr pipe: %v", r)
			}
			wg.Done()
		}()
		defer close(doneStderr)
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			go fmt.Fprintln(stdin, "yes\n")
			text := scanner.Text()
			if strings.Contains(text, "[yes/no]") {
				continue
			} else {
				select {
				case logChan <- LogEntry{Message: text, IsError: true}:
				case <-doneStderr:
					return
				}
			}
		}
		if err := scanner.Err(); err != nil {
			logger.GetLogger().Errorf("Error reading stderr pipe: %v", err)
		}
	}()

	err = session.Start(command)
	if err != nil {
		logChan <- LogEntry{Message: "Pipeline Done", IsError: true}
		logger.GetLogger().Errorf("Failed to run SSH command: %s", err.Error())
		return err
	}

	err = session.Wait()
	if err != nil {
		logger.GetLogger().Errorf("SSH command execution failed: %s", err.Error())
		logChan <- LogEntry{Message: "Pipeline Done", IsError: true}
		return err
	}
	logChan <- LogEntry{Message: "Pipeline Done", IsError: false}
	close(logChan)
	return nil
}

func (executor *SSHExecutor) ExecuteCommandWithoutReturn(command string) error {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SSH session: %s", err.Error())
		return err
	}
	defer session.Close()
	err = session.Run(command)
	if err != nil {
		log.Printf("Failed to execute command: %s", err.Error())
		return err
	}
	return nil
}

func (executor *SSHExecutor) ExecuteCMDWithoutReturn(command string, outputHandler func(string)) error {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SSH session: %s", err.Error())
		return err
	}
	defer session.Close()
	err = session.Run(command)
	if err != nil {
		log.Printf("Failed to execute command: %s", err.Error())
		return err
	}
	outputHandler(fmt.Sprintf("Successfully to execute command:%s", command))
	return nil
}

// CopyFile copies a file over SSH using SCP.
func (executor *SSHExecutor) CopyFile(srcFile, destFile string, outputHandler func(string)) error {
	src, err := os.Open(srcFile)
	if err != nil {
		logger.GetLogger().Errorf("Failed to open source file: %s", err.Error())
		return err
	}
	defer src.Close()

	sftpClient, err := sftp.NewClient(executor.Connection.Client)
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SFTP client: %s", err.Error())
		return err
	}
	defer sftpClient.Close()

	dest, err := sftpClient.Create(destFile)
	if err != nil {
		logger.GetLogger().Errorf("Failed to create destination file: %s", err.Error())
		return err
	}
	defer dest.Close()

	if _, err := io.Copy(dest, src); err != nil {
		logger.GetLogger().Errorf("Failed to copy file: %s", err.Error())
		return err
	}

	outputHandler(fmt.Sprintf("Copied file %s to %s", srcFile, destFile))
	return nil
}

func (executor *SSHExecutor) MkDirALL(path string, outputHandler func(string)) error {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SSH session: %s", err.Error())
		return err
	}
	defer session.Close()
	path = filepath.ToSlash(path)
	cmd := fmt.Sprintf("mkdir -p %s", path)
	err = session.Run(cmd)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create directory '%s' on remote host: %s", path, err)
		log.Println("%s: %s", errMsg, err.Error())
		return err
	}
	_ = fmt.Sprintf("Directory '%s' created successfully on remote host\n", path)
	outputHandler(fmt.Sprintf("Mkdir Directory: %s", path))
	return nil
}

func (executor *SSHExecutor) AddHosts(record entity.Record, outputHandler func(string)) error {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SSH session: %s", err.Error())
		return err
	}
	defer session.Close()
	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	err = session.Run("cat /etc/hosts")
	if err != nil {
		errMsg := fmt.Errorf("failed to read /etc/hosts: %s: %w", stderr.String(), err)
		log.Println("%s: %s", errMsg, err.Error())
		return err
	}
	if strings.TrimSpace(stdout.String()) != "" {
		cmdUpdate := fmt.Sprintf(`
		#!/bin/bash
        # Remove all lines containing the hostname
        sudo sed -i "/^.* %s$/d" /etc/hosts
        # Add new entry
        echo "%s %s" | sudo tee -a /etc/hosts > /dev/null
    `, record.Domain, record.IP, record.Domain)
		_, err = executor.ExecuteShortCommand(cmdUpdate)
		if err != nil {
			logger.GetLogger().Errorf("failed to update /etc/hosts: %v", err)
			return fmt.Errorf("failed to update /etc/hosts: %v", err)
		}
		logger.GetLogger().Infof("Updated %s to IP %s\n", record.Domain, record.IP)
		fmt.Printf("Updated %s to IP %s\n", record.Domain, record.IP)
	} else {
		cmdAdd := fmt.Sprintf(`echo "%s %s" >> /etc/hosts`, record.IP, record.Domain)
		_, err = executor.ExecuteShortCommand(cmdAdd)
		if err != nil {
			logger.GetLogger().Errorf("failed to add to /etc/hosts: %v", err)
			return fmt.Errorf("failed to add to /etc/hosts: %v", err)
		}
		logger.GetLogger().Infof("Added %s with IP %s\n", record.Domain, record.IP)
		fmt.Printf("Added %s with IP %s\n", record.Domain, record.IP)
	}
	outputHandler(fmt.Sprintf("Add Hosts"))
	return nil
}
