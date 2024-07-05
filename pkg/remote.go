package pkg

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type SSHExecutor struct {
	Connection SSHConnection // SSH连接信息
}

type Executor interface {
	ExecuteCommand(command string, logChan chan LogEntry) error          // 执行命令
	CopyFile(srcFile, destFile string, outputHandler func(string)) error // 复制文件
}

func NewSSHExecutor(connection SSHConnection) *SSHExecutor {
	return &SSHExecutor{Connection: connection}
}

func (executor *SSHExecutor) ExecuteCommand(command string, logChan chan LogEntry) error {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		return fmt.Errorf("Failed to create SSH session: %s", err)
	}
	defer session.Close()

	cmd := exec.Command("sh", "-c", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stdout for SSH command: %v", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("Failed to create stderr pipe: %s", err)
	}

	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			logChan <- LogEntry{Message: scanner.Text(), IsError: false}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			logChan <- LogEntry{Message: scanner.Text(), IsError: true}
		}
	}()

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("Failed to run SSH command: %s", err)
	}

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("SSH command execution failed: %s", err)
	}
	return nil
}

func (executor *SSHExecutor) CopyFile(srcFile, destFile string, outputHandler func(string)) error {
	src, err := os.Open(srcFile)
	if err != nil {
		return fmt.Errorf("Failed to open source file: %s", err)
	}
	defer src.Close()

	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		return fmt.Errorf("Failed to create SSH session: %s", err)
	}
	defer session.Close()

	dest, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("Failed to setup stdin for SCP: %s", err)
	}

	go func() {
		srcStat, err := src.Stat()
		if err != nil {
			fmt.Printf("Failed to get source file info: %s\n", err)
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

func (executor *SSHExecutor) CopyRemoteToRemote(srcHost, srcFile, destHost, destFile string, outputHandler func(string)) error {
	// 使用 SCP 将文件从源主机复制到目标主机
	cmd := fmt.Sprintf("/usr/bin/scp -o StrictHostKeyChecking=no %s:%s %s:%s", srcHost, srcFile, destHost, destFile)
	outputHandler(fmt.Sprintf("Executing command: %s", cmd))

	// 执行远程命令
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		return fmt.Errorf("Failed to create SSH session: %s", err)
	}
	defer session.Close()
	err = session.Run(cmd)
	if err != nil {
		return fmt.Errorf("Failed to run command: %s", err)
	}

	outputHandler(fmt.Sprintf("Copied file %s from %s to %s on %s", srcFile, srcHost, destFile, destHost))
	return nil
}
