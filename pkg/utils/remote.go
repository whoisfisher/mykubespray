package utils

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// SSHExecutor implements Executor for SSH connections.
type SSHExecutor struct {
	Connection SSHConnection
}

// NewSSHExecutor creates a new instance of SSHExecutor.
func NewSSHExecutor(connection SSHConnection) *SSHExecutor {
	return &SSHExecutor{Connection: connection}
}

// ExecuteCommand executes a command over SSH.

func (executor *SSHExecutor) ExecuteShortCommand(command string) (string, error) {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		log.Printf("Failed to create SSH session: %s", err.Error())
		return "", err
	}
	defer session.Close()
	res, err := session.CombinedOutput(command)
	if err != nil {
		log.Printf("Failed to create SSH session: %s", err.Error())
		return "", err
	}
	return string(res), nil
}

func (executor *SSHExecutor) ExecuteCommand(command string, logChan chan LogEntry) error {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		log.Printf("Failed to create SSH session: %s", err.Error())
		return err
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		log.Printf("Unable to setup stdin for session: %v", err)
		return err
	}

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		log.Printf("Unable to create stdout pipe: %v", err.Error())
		return err
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		log.Printf("Failed to create stderr pipe: %s", err.Error())
		return err
	}

	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			fmt.Fprintln(stdin, "yes\n")
			text := scanner.Text()
			logChan <- LogEntry{Message: text, IsError: false}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			fmt.Fprintln(stdin, "yes\n")
			text := scanner.Text()
			logChan <- LogEntry{Message: text, IsError: true}
		}
	}()

	err = session.Start(command)
	if err != nil {
		log.Printf("Failed to run SSH command: %s", err.Error())
		return err
	}

	err = session.Wait()
	if err != nil {
		log.Printf("SSH command execution failed: %s", err.Error())
		return err
	}
	return nil
}

func (executor *SSHExecutor) ExecuteCommandWithTTY(command string, logChan chan LogEntry) error {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		log.Printf("Failed to create SSH session: %s", err.Error())
		return err
	}
	defer session.Close()

	// 设置标准输入、输出和错误
	stdin, err := session.StdinPipe()
	if err != nil {
		log.Printf("Unable to setup stdin for session: %v", err)
		return err
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		log.Printf("Unable to setup stdout for session: %v", err)
		return err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		log.Printf("Unable to setup stderr for session: %v", err)
		return err
	}

	// 启动 shell
	err = session.Shell()
	if err != nil {
		log.Printf("Failed to start shell: %s", err.Error())
		return err
	}

	// 使用 bufio.Scanner 读取输出
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			text := scanner.Text()
			logChan <- LogEntry{Message: text, IsError: false}
			if strings.Contains(text, "Are you sure to delete this") && strings.Contains(text, "? [yes/no]: ") {
				// 发送 "yes" 到标准输入
				_, err := fmt.Fprintln(stdin, "yes")
				if err != nil {
					log.Printf("Failed to send 'yes' to stdin: %v", err)
					return
				}
			}
		}
	}()

	// 同样读取 stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			text := scanner.Text()
			logChan <- LogEntry{Message: text, IsError: true}
		}
	}()

	// 执行命令
	err = session.Run(command)
	if err != nil {
		log.Printf("Failed to run SSH command: %s", err.Error())
		return err
	}

	// 等待命令完成
	wg.Wait()

	return nil
}

func (executor *SSHExecutor) ExecuteCommandWithoutReturn(command string) error {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		log.Printf("Failed to create SSH session: %s", err.Error())
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

// CopyFile copies a file over SSH using SCP.
func (executor *SSHExecutor) CopyFile(srcFile, destFile string, outputHandler func(string)) error {
	src, err := os.Open(srcFile)
	if err != nil {
		log.Printf("Failed to open source file: %s", err.Error())
		return err
	}
	defer src.Close()

	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		log.Printf("Failed to create SSH session: %s", err.Error())
		return err
	}
	defer session.Close()

	dest, err := session.StdinPipe()
	if err != nil {
		log.Printf("Failed to setup stdin for SCP: %s", err.Error())
		return err
	}

	go func() {
		srcStat, err := src.Stat()
		if err != nil {
			log.Printf("Failed to get source file info: %s\n", err)
			return
		}
		defer dest.Close()

		fmt.Fprintln(dest, "C0644", srcStat.Size(), destFile)
		io.Copy(dest, src)
		fmt.Fprint(dest, "\x00")
	}()

	outputHandler(fmt.Sprintf("Copied file %s to %s", srcFile, destFile))
	return nil
}

func (executor *SSHExecutor) MkDirALL(path string, outputHandler func(string)) error {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		log.Printf("Failed to create SSH session: %s", err.Error())
		return err
	}
	defer session.Close()
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
