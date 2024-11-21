package utils

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/pkg/sftp"
	"github.com/whoisfisher/mykubespray/pkg/entity"
	"github.com/whoisfisher/mykubespray/pkg/logger"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SSHExecutor implements Executor for SSH connections.
type SSHExecutor struct {
	Connection SSHConnection
	Host       entity.Host
}

func NewExecutor(host entity.Host) *SSHExecutor {
	connection, err := NewConnection(host)
	if err != nil {
		return nil
	}
	return &SSHExecutor{Connection: *connection, Host: host}
}

// NewSSHExecutor creates a new instance of SSHExecutor.
func NewSSHExecutor(connection SSHConnection) *SSHExecutor {
	return &SSHExecutor{Connection: connection}
}

// ExecuteCommand executes a command over SSH.

func (executor *SSHExecutor) ExecuteShortCommand(command string) (string, error) {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SSH session: %v", err)
		return "", err
	}
	defer session.Close()
	res, err := session.CombinedOutput(command)
	if err != nil {
		logger.GetLogger().Errorf("Failed to execute command: %v, %s", err, res)
		return "", err
	}
	return string(res), nil
}

func (executor *SSHExecutor) ExecuteShortCMD(command string) ([]byte, error) {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SSH session: %v", err)
		return nil, err
	}
	defer session.Close()
	res, err := session.CombinedOutput(command)
	if err != nil {
		logger.GetLogger().Errorf("Failed to execute command: %v", err)
		return nil, err
	}
	return res, nil
}

func (executor *SSHExecutor) ExecuteCommandWithoutReturn(command string) error {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SSH session: %v", err)
		return err
	}
	defer session.Close()
	err = session.Run(command)
	if err != nil {
		logger.GetLogger().Errorf("Failed to execute command: %v", err)
		return err
	}
	return nil
}

func (executor *SSHExecutor) ExecuteCMDWithoutReturn(command string, outputHandler func(string)) error {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SSH session: %v", err)
		return err
	}
	defer session.Close()
	err = session.Run(command)
	if err != nil {
		logger.GetLogger().Errorf("Failed to execute command: %v", err)
		return err
	}
	outputHandler(fmt.Sprintf("Successfully to execute command:%s", command))
	return nil
}

func (executor *SSHExecutor) WhoAmI() string {
	command := fmt.Sprintf("whoami")
	user, err := executor.ExecuteShortCommand(command)
	if err != nil {
		logger.GetLogger().Warnf("Read username failed: %v", err)
		return ""
	}
	return strings.TrimSpace(user)
}

func (executor *SSHExecutor) ExecuteCommand(command string, logChan chan LogEntry) error {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SSH session: %v", err)
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
		logger.GetLogger().Errorf("Unable to create stdout pipe: %v", err)
		return err
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create stderr pipe: %v", err)
		return err
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.GetLogger().Errorf("Recovered from panic in stderr pipe: %v", r)
			}
		}()
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
		defer func() {
			if r := recover(); r != nil {
				logger.GetLogger().Errorf("Recovered from panic in stderr pipe: %v", r)
			}
		}()
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
		logger.GetLogger().Errorf("Failed to run SSH command: %v", err)
		return err
	}

	err = session.Wait()
	if err != nil {
		logger.GetLogger().Errorf("SSH command execution failed: %v", err)
		logChan <- LogEntry{Message: "Pipeline Done", IsError: true}
		return err
	}
	logChan <- LogEntry{Message: "Pipeline Done", IsError: false}
	return nil
}

func (executor *SSHExecutor) ExecuteCommandNew(command string, logChan chan LogEntry) error {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SSH session: %v", err)
		logChan <- LogEntry{Message: "Pipeline Failed", IsError: true}
		return err
	}
	defer session.Close()
	//session.RequestPty("xterm", 80, 40, ssh.TerminalModes{})

	stdin, err := session.StdinPipe()
	if err != nil {
		logger.GetLogger().Errorf("Unable to setup stdin for session: %v", err)
		logChan <- LogEntry{Message: "Pipeline Failed", IsError: true}
		return err
	}

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		logger.GetLogger().Errorf("Unable to create stdout pipe: %v", err)
		logChan <- LogEntry{Message: "Pipeline Failed", IsError: true}
		return err
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create stderr pipe: %v", err)
		logChan <- LogEntry{Message: "Pipeline Failed", IsError: true}
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
		logger.GetLogger().Errorf("Failed to run SSH command: %v", err)
		logChan <- LogEntry{Message: "Pipeline Failed", IsError: true}
		return err
	}

	err = session.Wait()
	if err != nil {
		logger.GetLogger().Errorf("SSH command execution failed: %v", err)
		logChan <- LogEntry{Message: "Pipeline Failed", IsError: true}
		return err
	}
	logChan <- LogEntry{Message: "Pipeline Success", IsError: false}
	close(logChan)
	return nil
}

func (executor *SSHExecutor) CopyMultiFile(files []entity.FileSrcDest, outputHandler func(string)) *CopyResult {
	var wg sync.WaitGroup
	results := make(chan MachineResult, len(files))
	for _, file := range files {
		wg.Add(1)
		go func(file entity.FileSrcDest) {
			defer wg.Done()
			byteData, err := os.ReadFile(file.SrcFile)
			if err != nil {
				logger.GetLogger().Errorf("Failed to open source file: %v", err)
				results <- MachineResult{Machine: "", Success: false, Error: fmt.Sprintf("Failed to open source file: %v", err)}
				return
			}

			fileName := filepath.Base(file.DestFile)
			tempFile := fmt.Sprintf("/tmp/%s", fileName)

			touchCommand := fmt.Sprintf("echo '%s' > %s", string(byteData), tempFile)
			err = executor.ExecuteCommandWithoutReturn(touchCommand)
			if err != nil {
				logger.GetLogger().Errorf("Failed to copy file to destination: %v", err)
				return
			}

			command := fmt.Sprintf("cp -f %s %s", tempFile, file.DestFile)
			if executor.WhoAmI() != "root" {
				command = SudoPrefixWithPassword(command, executor.Host.Password)
			}
			err = executor.ExecuteCommandWithoutReturn(command)
			if err != nil {
				logger.GetLogger().Errorf("Failed to copy file to destination: %v", err)
				results <- MachineResult{Machine: "", Success: false, Error: fmt.Sprintf("Failed to copy file to destination: %v", err)}
				return
			}

			rmCommand := fmt.Sprintf("rm -f %s", tempFile)
			err = executor.ExecuteCommandWithoutReturn(rmCommand)
			if err != nil {
				logger.GetLogger().Errorf("Failed to delete temp file: %v", err)
				return
			}

			results <- MachineResult{Machine: "", Success: true, Error: ""}
			outputHandler(fmt.Sprintf("Copied file %s to %s", file.SrcFile, file.DestFile))
			return
		}(file)
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	var successCount, failureCount int
	var copyResult CopyResult
	var machineResults []MachineResult
	for result := range results {
		if result.Success {
			logger.GetLogger().Infof("Successfully copied file to %s\n", result.Machine)
			successCount++
		} else {
			logger.GetLogger().Errorf("Failed to copy file to %s: %s\n", result.Machine, result.Error)
			failureCount++
		}
		machineResults = append(machineResults, result)
	}
	copyResult.Results = machineResults
	if failureCount > 0 {
		copyResult.OverallSuccess = false
	} else {
		copyResult.OverallSuccess = true
	}
	return &copyResult
}

// CopyFile copies a file over SSH using SCP.
func (executor *SSHExecutor) CopyFile(srcFile, destFile string, outputHandler func(string)) error {
	byteData, err := os.ReadFile(srcFile)
	if err != nil {
		logger.GetLogger().Errorf("Failed to open source file: %v", err)
		return err
	}

	fileName := filepath.Base(destFile)
	tempFile := fmt.Sprintf("/tmp/%s", fileName)

	touchCommand := fmt.Sprintf("echo '%s' > %s", string(byteData), tempFile)
	err = executor.ExecuteCommandWithoutReturn(touchCommand)
	if err != nil {
		logger.GetLogger().Errorf("Failed to copy file to destination: %v", err)
		return err
	}

	command := fmt.Sprintf("cp -f %s %s", tempFile, destFile)
	if executor.WhoAmI() != "root" {
		command = SudoPrefixWithPassword(command, executor.Host.Password)
	}
	err = executor.ExecuteCommandWithoutReturn(command)
	if err != nil {
		logger.GetLogger().Errorf("Failed to copy file to destination: %v", err)
		return err
	}

	rmCommand := fmt.Sprintf("rm -f %s", tempFile)
	err = executor.ExecuteCommandWithoutReturn(rmCommand)
	if err != nil {
		logger.GetLogger().Errorf("Failed to delete temp file: %v", err)
		return err
	}

	outputHandler(fmt.Sprintf("Copied file %s to %s", srcFile, destFile))
	return nil
}

func (executor *SSHExecutor) MkDirALL(path string, outputHandler func(string)) error {
	path = filepath.ToSlash(path)
	command := fmt.Sprintf("mkdir -p %s", path)
	if executor.WhoAmI() != "root" {
		command = SudoPrefixWithPassword(command, executor.Host.Password)
	}
	err := executor.ExecuteCommandWithoutReturn(command)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create directory '%s' on remote host: %v", path, err)
		logger.GetLogger().Errorf("%s: %v", errMsg, err)
		return err
	}
	_ = fmt.Sprintf("Directory '%s' created successfully on remote host\n", path)
	outputHandler(fmt.Sprintf("Mkdir Directory: %s", path))
	return nil
}

func (executor *SSHExecutor) AddHosts(record entity.Record, outputHandler func(string)) error {
	getHostContentCMD := "cat /etc/hosts"
	hostContent, err := executor.ExecuteShortCommand(getHostContentCMD)
	if err != nil {
		errMsg := fmt.Errorf("failed to read /etc/hosts: %w", err)
		log.Println("%s: %v", errMsg, err)
		return err
	}
	if strings.TrimSpace(hostContent) != "" {
		cmdUpdate := fmt.Sprintf(`
		#!/bin/bash
        # Remove all lines containing the hostname
        sudo sed -i "/^.* %s$/d" /etc/hosts
        # Add new entry
        echo "%s %s" | sudo tee -a /etc/hosts > /dev/null
    `, record.Domain, record.IP, record.Domain)
		cmdUpdate = fmt.Sprintf("bash -c '%s'", cmdUpdate)
		if executor.WhoAmI() != "root" {
			cmdUpdate = SudoPrefixWithPassword(cmdUpdate, executor.Host.Password)
		}
		_, err = executor.ExecuteShortCommand(cmdUpdate)
		if err != nil {
			logger.GetLogger().Errorf("failed to update /etc/hosts: %v", err)
			return fmt.Errorf("failed to update /etc/hosts: %w", err)
		}
		logger.GetLogger().Infof("Updated %s to IP %s\n", record.Domain, record.IP)
		fmt.Printf("Updated %s to IP %s\n", record.Domain, record.IP)
	} else {
		cmdAdd := fmt.Sprintf(`bash -c 'echo "%s %s" >> /etc/hosts'`, record.IP, record.Domain)
		if executor.WhoAmI() != "root" {
			cmdAdd = SudoPrefixWithPassword(cmdAdd, executor.Host.Password)
		}
		_, err = executor.ExecuteShortCommand(cmdAdd)
		if err != nil {
			logger.GetLogger().Errorf("failed to add to /etc/hosts: %v", err)
			return fmt.Errorf("failed to add to /etc/hosts: %w", err)
		}
		logger.GetLogger().Infof("Added %s with IP %s\n", record.Domain, record.IP)
		fmt.Printf("Added %s with IP %s\n", record.Domain, record.IP)
	}
	outputHandler(fmt.Sprintf("Add Hosts"))
	return nil
}

func (executor *SSHExecutor) AddMultiHosts(records []entity.Record, outputHandler func(string)) error {
	getHostContentCMD := "cat /etc/hosts"
	hostContent, err := executor.ExecuteShortCommand(getHostContentCMD)
	if err != nil {
		errMsg := fmt.Errorf("failed to read /etc/hosts: %w", err)
		log.Println("%s: %s", errMsg, err.Error())
		return err
	}
	lines := strings.Split(hostContent, "\n")
	var updateContent strings.Builder
	domainMap := make(map[string]string)

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			ip, domain := parts[0], parts[1]
			domainMap[domain] = ip
		}
	}

	for _, record := range records {
		domainMap[record.Domain] = record.IP
	}

	for key, value := range domainMap {
		updateContent.WriteString(fmt.Sprintf("%s %s\n", value, key))
	}

	tmpFile := "/tmp/hosts"
	err = os.WriteFile(tmpFile, []byte(updateContent.String()), 0644)
	if err != nil {
		logger.GetLogger().Error("Failed to write "+
			""+
			" temporary file: %s", err)
		return fmt.Errorf("Failed to write to temporary file: %w", err)
	}
	cmd := fmt.Sprintf("cp %s /etc/hosts", tmpFile)
	if executor.WhoAmI() != "root" {
		cmd = SudoPrefixWithPassword(cmd, executor.Host.Password)
	}
	_, err = executor.ExecuteShortCommand(cmd)
	if err != nil {
		logger.GetLogger().Errorf("failed to add to /etc/hosts: %v", err)
		return fmt.Errorf("failed to add to /etc/hosts: %w", err)
	}
	outputHandler(fmt.Sprintf("Add Hosts"))
	return nil
}

func (executor *SSHExecutor) UpdateHostsFile(ip string, domain string) error {
	getHostContentCMD := "cat /etc/hosts"
	hostContent, err := executor.ExecuteShortCommand(getHostContentCMD)
	if err != nil {
		logger.GetLogger().Errorf("读取 /etc/hosts 出错: %v\", err")
		return fmt.Errorf("读取 /etc/hosts 出错: %w", err)
	}
	lines := strings.Split(hostContent, "\n")
	domainExists := false
	var updatedLines []string

	for _, line := range lines {
		if strings.Contains(line, domain) {
			domainExists = true
			parts := strings.Fields(line)
			if len(parts) > 0 {
				updatedLines = append(updatedLines, fmt.Sprintf("%s %s", ip, domain))
			}
		} else {
			updatedLines = append(updatedLines, line)
		}
	}

	if !domainExists {
		updatedLines = append(updatedLines, fmt.Sprintf("%s %s", ip, domain))
	}

	// 写入更新后的内容
	command := fmt.Sprintf("bash -c \"echo -n '%s' | sudo tee /etc/hosts\"", strings.Join(updatedLines, "\n"))
	if executor.WhoAmI() != "root" {
		command = SudoPrefixWithPassword(command, executor.Host.Password)
	}
	_, err = executor.ExecuteShortCommand(command)
	if err != nil {
		logger.GetLogger().Errorf("写入 /etc/hosts 出错: %v", err)
		return fmt.Errorf("写入 /etc/hosts 出错: %w", err)
	}
	return nil
}

func (executor *SSHExecutor) UpdateResolvFile(ip string) error {
	getDNSContentCMD := "cat /etc/resolv.conf"
	dnsContent, err := executor.ExecuteShortCommand(getDNSContentCMD)
	if err != nil {
		logger.GetLogger().Errorf("读取 /etc/resolv.conf 出错: %v", err)
		return fmt.Errorf("读取 /etc/resolv.conf 出错: %w", err)
	}

	lines := strings.Split(dnsContent, "\n")
	ipExists := false

	for _, line := range lines {
		if strings.Contains(line, ip) {
			ipExists = true
			break
		}
	}

	if !ipExists {
		command := fmt.Sprintf("bash -c \" echo -n 'nameserver %s\n' | sudo tee -a /etc/resolv.conf\"", ip)
		if executor.WhoAmI() != "root" {
			command = SudoPrefixWithPassword(command, executor.Host.Password)
		}
		_, err = executor.ExecuteShortCommand(command)
		if err != nil {
			logger.GetLogger().Errorf("追加到 /etc/resolv.conf 出错: %v", err)
			return fmt.Errorf("追加到 /etc/resolv.conf 出错: %w", err)
		}
	} else {
		logger.GetLogger().Infof("IP 已存在，跳过追加")
		fmt.Println("IP 已存在，跳过追加")
	}
	return nil
}

func (executor *SSHExecutor) ChangeExpiredPassword(currentPassword, newPassword string) error {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SSH session: %v", err)
		return err
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		logger.GetLogger().Errorf("Unable to setup stdin for session: %v", err)
		return err
	}

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		logger.GetLogger().Errorf("Unable to create stdout pipe: %v", err)
		return err
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create stderr pipe: %v", err)
		return err
	}

	outputReader := io.MultiReader(stdoutPipe, stderrPipe)

	if err := session.RequestPty("xterm", 80, 40, ssh.TerminalModes{}); err != nil {
		return fmt.Errorf("Cannot request tty: %w", err)
	}

	if err := session.Shell(); err != nil {
		return fmt.Errorf("Cannot start shell: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // 设置超时为30秒
	defer cancel()

	// 创建一个通道用于接收扫描结果
	resultCh := make(chan error)

	go func() {
		scanner := bufio.NewScanner(outputReader)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)

			if strings.Contains(line, "password has expired") || strings.Contains(line, "You must change your password") {
				fmt.Println("Password expired，updating password...")

				if _, err := fmt.Fprintln(stdin, currentPassword); err != nil {
					resultCh <- fmt.Errorf("Cannot input current password: %w", err)
					return
				}
				time.Sleep(1 * time.Second)

				if _, err := fmt.Fprintln(stdin, newPassword); err != nil {
					resultCh <- fmt.Errorf("Cannot input new password: %w", err)
					return
				}
				time.Sleep(1 * time.Second)

				if _, err := fmt.Fprintln(stdin, newPassword); err != nil {
					resultCh <- fmt.Errorf("Cannot confirm new password: %w", err)
					return
				}

				fmt.Println("Update password success")
				resultCh <- nil // 表示成功
				return
			}
		}
		resultCh <- scanner.Err() // 发送扫描错误
	}()

	// 等待扫描结果或超时
	select {
	case err := <-resultCh: // 从通道接收结果
		if err != nil {
			return fmt.Errorf("Read input/output error: %w", err)
		}
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("timeout: No password expiration warning detected.")
		}
	}

	return nil
}

// CheckPasswordInfo 获取用户的密码信息
func (executor *SSHExecutor) CheckPasswordInfo() (*entity.PasswordInfo, error) {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		return &entity.PasswordInfo{}, fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	command := fmt.Sprintf("chage -l %s", executor.Host.User)
	if executor.WhoAmI() != "root" {
		command = SudoPrefixWithPassword(command, executor.Host.Password)
	}

	output, err := session.CombinedOutput(command)
	if err != nil {
		return &entity.PasswordInfo{}, fmt.Errorf("failed to execute command: %w, %s", err, string(output))
	}

	info := &entity.PasswordInfo{}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		logger.GetLogger().Infof(line)

		if strings.Contains(line, "Last password change") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				info.LastChange = strings.TrimSpace(parts[len(parts)-1])
			}
		} else if strings.Contains(line, "Password expires") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				info.PasswordExpires = strings.TrimSpace(parts[len(parts)-1])
			}
		} else if strings.Contains(line, "Password inactive") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				info.PasswordInactive = strings.TrimSpace(parts[len(parts)-1])
			}
		} else if strings.Contains(line, "Account expires") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				info.AccountExpires = strings.TrimSpace(parts[len(parts)-1])
			}
		} else if strings.Contains(line, "Minimum number of days between password change") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				fmt.Sscanf(strings.TrimSpace(strings.Split(line, ":")[len(parts)-1]), "%d", &info.MinDays)
			}
		} else if strings.Contains(line, "Maximum number of days between password change") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				fmt.Sscanf(strings.TrimSpace(strings.Split(line, ":")[len(parts)-1]), "%d", &info.MaxDays)
			}
		} else if strings.Contains(line, "Number of days of warning before password expires") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				fmt.Sscanf(strings.TrimSpace(strings.Split(line, ":")[len(parts)-1]), "%d", &info.WarningDays)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return &entity.PasswordInfo{}, fmt.Errorf("Input/Output read error: %w", err)
	}

	return info, nil
}

// UpdatePassword 修改用户密码
func (executor *SSHExecutor) UpdatePassword(currentPassword, newPassword string) error {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SSH session: %s", err.Error())
		return err
	}
	defer session.Close()

	command := ""
	if executor.WhoAmI() == "root" {
		command = fmt.Sprintf("echo '%s:%s' | chpasswd", "root", newPassword)
	} else {
		command = fmt.Sprintf("bash -c \"echo '%s:%s' | sudo chpasswd\"", executor.Host.User, newPassword)
		command = SudoPrefixWithPassword(command, executor.Host.Password)
	}

	output, err := session.CombinedOutput(command)
	if err != nil {
		return fmt.Errorf("failed to change password: %w, output: %s", err, output)
	}

	return nil
}

func (executor *SSHExecutor) ReadFile(path string) ([]byte, error) {
	cmd := fmt.Sprintf("cat %s", path)
	data, err := executor.ExecuteShortCMD(cmd)
	if err != nil {
		logger.GetLogger().Errorf("Read %s failed: %v", path, err)
		return nil, err
	}
	return data, nil
}

func (executor *SSHExecutor) WriteFile(content []byte, path string, perm os.FileMode) error {
	cmd := fmt.Sprintf("bash -c \"echo -e '%s' > %s\"", content, path)
	if executor.WhoAmI() != "root" {
		cmd = SudoPrefixWithPassword(cmd, executor.Host.Password)
	}
	err := executor.ExecuteCommandWithoutReturn(cmd)
	if err != nil {
		logger.GetLogger().Errorf("Write %s failed: %v", path, err)
		return err
	}
	chmodCommand := fmt.Sprintf("bash -c \"chmod %s %s\"", perm, path)
	if executor.WhoAmI() != "root" {
		chmodCommand = SudoPrefixWithPassword(chmodCommand, executor.Host.Password)
	}
	err = executor.ExecuteCommandWithoutReturn(chmodCommand)
	if err != nil {
		logger.GetLogger().Errorf("Chmod %s failed: %v", path, err)
		return err
	}
	return nil
}

func (executor *SSHExecutor) GenerateSSHKey() (privateKey, publicKey string, err error) {
	return "", "", nil
}

func (executor *SSHExecutor) WritePrivateKey(privateKey string) error {
	if !executor.DirIsExist("~/.ssh") {
		err := executor.MkDirALL("~/.ssh", func(s string) {})
		if err != nil {
			logger.GetLogger().Errorf("Create directory ~/.ssh failed: %v", err)
			return err
		}
	}
	err := executor.WriteFile([]byte(privateKey), "~/.ssh/id_rsa", 0600)
	if err != nil {
		logger.GetLogger().Errorf("Failed to save private key to file: %v", err)
		return err
	}
	logger.GetLogger().Infof("Private key saved to id_rsa file")
	return nil
}

func (executor *SSHExecutor) SetupPasswordLessLogin(pubkey string) error {
	cmd := fmt.Sprintf("echo \"%s\" >> ~/.ssh/authorized_keys", pubkey)
	err := executor.ExecuteCommandWithoutReturn(cmd)
	if err != nil {
		logger.GetLogger().Errorf("failed to add public key to authorized_keys: %v", err)
		return err
	}
	logger.GetLogger().Infof("Public key added to remote host for passwordless login")
	return nil
}

func (executor *SSHExecutor) DirIsExist(path string) bool {
	cmd := fmt.Sprintf("test -d %s && echo 'exists' || echo 'not exists'", path)
	output, err := executor.ExecuteShortCommand(cmd)
	if err != nil {
		return false
	} else if string(output) == "exists\n" {
		return true
	}
	return false
}

func (executor *SSHExecutor) FileIsExists(path string) bool {
	cmd := fmt.Sprintf("test -f %s && echo 'exists' || echo 'not exists'", path)
	output, err := executor.ExecuteShortCommand(cmd)
	if err != nil {
		return false
	} else if string(output) == "exists\n" {
		return true
	}
	return false
}

func (executor *SSHExecutor) FetchFile(path string, local string, perm os.FileMode) error {
	sftpClient, err := sftp.NewClient(executor.Connection.Client)
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SFTP client: %v", err)
		return fmt.Errorf("Failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	remoteFile, err := sftpClient.Open(path)
	if err != nil {
		logger.GetLogger().Errorf("Failed to open remote file: %v", err)
		return fmt.Errorf("Failed to open remote file: %w", err)
	}
	defer remoteFile.Close()

	localFile, err := os.Create(local)
	if err != nil {
		logger.GetLogger().Errorf("Failed to create local file: %v", err)
		return fmt.Errorf("Failed to create local file: %w", err)
	}
	defer localFile.Close()

	if err := localFile.Chmod(perm); err != nil {
		logger.GetLogger().Errorf("Failed to set file permissions: %v", err)
		return fmt.Errorf("Failed to set file permissions: %w", err)
	}

	// 使用缓冲区流式读取和写入
	buffer := make([]byte, 1024*16) // 16KB 缓冲区
	for {
		n, err := remoteFile.Read(buffer)
		if err != nil && err != io.EOF {
			logger.GetLogger().Errorf("Error reading remote file: %v", err)
			return fmt.Errorf("Error reading remote file: %w", err)
		}
		if n == 0 {
			break // 文件读取完毕
		}

		// 将读取的数据写入本地文件
		_, err = localFile.Write(buffer[:n])
		if err != nil {
			logger.GetLogger().Errorf("Error writing to local file: %v", err)
			return fmt.Errorf("Error writing to local file: %w", err)
		}
	}

	// 成功传输文件
	logger.GetLogger().Infof("Successfully fetched file from %s to %s", path, local)
	return nil
}

func (executor *SSHExecutor) Upload(localFile, remoteFile string) error {

	sftpClient, err := sftp.NewClient(executor.Connection.Client)
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SFTP client: %v", err)
		return fmt.Errorf("Failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()
	var tempPath string
	if executor.WhoAmI() != "root" {
		tempPath = filepath.Join("/tmp/", filepath.Base(localFile))
		tempPath = filepath.ToSlash(tempPath)
	} else {
		tempPath = remoteFile
	}

	srcFile, err := os.Open(localFile)
	if err != nil {
		logger.GetLogger().Errorf("Failed to open local file %s: %v", localFile, err)
		return fmt.Errorf("Failed to open local file %s: %w", localFile, err)
	}
	defer srcFile.Close()

	destFile, err := sftpClient.Create(tempPath)
	if err != nil {
		logger.GetLogger().Errorf("Failed to create remote file %s: %v", tempPath, err)
		return fmt.Errorf("failed to create remote file %s: %w", tempPath, err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		logger.GetLogger().Errorf("Failed to copy file %s to %s: %v", srcFile, destFile, err)
		return fmt.Errorf("Failed to copy file %s to %s: %w", srcFile, destFile, err)
	}

	if executor.WhoAmI() != "root" {
		command := fmt.Sprintf("cp -f %s %s", tempPath, remoteFile)
		command = SudoPrefixWithPassword(command, executor.Host.Password)
		err := executor.ExecuteCommandWithoutReturn(command)
		if err != nil {
			logger.GetLogger().Errorf("Failed to copy %s to %s: %w", tempPath, remoteFile, err)
			return fmt.Errorf("Failed to copy %s to %s: %w", tempPath, remoteFile, err)
		}
	}
	logger.GetLogger().Infof("Successfully to upload file %s to %s", localFile, remoteFile)
	return nil
}

func (executor *SSHExecutor) Download(remoteFile, localFile string) error {
	sftpClient, err := sftp.NewClient(executor.Connection.Client)
	if err != nil {
		logger.GetLogger().Errorf("Failed to create SFTP client: %v", err)
		return fmt.Errorf("Failed to create SFTP client: %w", err)
	}
	var tempPath string
	if executor.WhoAmI() != "root" {
		tempPath = filepath.Join("/tmp/", filepath.Base(remoteFile))
		command := fmt.Sprintf("cp -f %s %s", remoteFile, tempPath)
		command = SudoPrefixWithPassword(command, executor.Host.Password)
		err := executor.ExecuteCommandWithoutReturn(command)
		if err != nil {
			logger.GetLogger().Errorf("Failed to copy %s to %s: %v", remoteFile, tempPath, err)
			return fmt.Errorf("Failed to copy %s to %s: %w", remoteFile, tempPath, err)
		}
	} else {
		tempPath = remoteFile
	}
	srcFile, err := sftpClient.Open(remoteFile)
	if err != nil {
		logger.GetLogger().Errorf("Failed to open remote file %s: %v", tempPath, err)
		return fmt.Errorf("Failed to open remote file %s: %w", tempPath, err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(localFile)
	if err != nil {
		logger.GetLogger().Errorf("Failed to create local file %s: %v", localFile, err)
		return fmt.Errorf("Failed to create local file %s: %w", localFile, err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		logger.GetLogger().Errorf("Failed to copy file %s to %s: %v", srcFile, destFile, err)
		return fmt.Errorf("Failed to copy file %s to %s: %w", srcFile, destFile, err)
	}

	logger.GetLogger().Infof("Successfully to download file %s to %s", remoteFile, localFile)
	return nil
}
