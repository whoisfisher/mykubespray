package pkg

import (
	"bufio"
	"fmt"
	"io"
	"os"
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
		return "", fmt.Errorf("failed to create SSH session: %s", err)
	}
	defer session.Close()
	res, err := session.CombinedOutput(command)
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %s", err)
	}
	return string(res), nil
}

func (executor *SSHExecutor) ExecuteCommand(command string, logChan chan LogEntry) error {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %s", err)
	}
	defer session.Close()

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("unable to setup stdout for SSH command: %v", err)
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %s", err)
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

	err = session.Start(command)
	if err != nil {
		return fmt.Errorf("failed to run SSH command: %s", err)
	}

	err = session.Wait()
	if err != nil {
		return fmt.Errorf("SSH command execution failed: %s", err)
	}
	return nil
}

func (executor *SSHExecutor) ExecuteCommandWithoutReturn(command string) error {
	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %s", err)
	}
	defer session.Close()
	err = session.Run(command)
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %s", err)
	}
	return nil
}

// CopyFile copies a file over SSH using SCP.
func (executor *SSHExecutor) CopyFile(srcFile, destFile string, outputHandler func(string)) error {
	src, err := os.Open(srcFile)
	if err != nil {
		return fmt.Errorf("failed to open source file: %s", err)
	}
	defer src.Close()

	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %s", err)
	}
	defer session.Close()

	dest, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to setup stdin for SCP: %s", err)
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

// CopyRemoteToRemote copies a file from one remote host to another using SCP.
func (executor *SSHExecutor) CopyRemoteToRemote(srcHost, srcFile, destHost, destFile string, outputHandler func(string)) error {
	cmd := fmt.Sprintf("/usr/bin/scp -o StrictHostKeyChecking=no %s:%s %s:%s", srcHost, srcFile, destHost, destFile)
	outputHandler(fmt.Sprintf("Executing command: %s", cmd))

	session, err := executor.Connection.Client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %s", err)
	}
	defer session.Close()

	err = session.Run(cmd)
	if err != nil {
		return fmt.Errorf("failed to run command: %s", err)
	}

	outputHandler(fmt.Sprintf("Copied file %s from %s to %s on %s", srcFile, srcHost, destFile, destHost))
	return nil
}
