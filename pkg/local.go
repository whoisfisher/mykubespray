package pkg

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// LocalExecutor implements Executor for local system commands.
type LocalExecutor struct{}

func NewLocalExecutor() *LocalExecutor {
	return &LocalExecutor{}
}

// ExecuteCommand executes a command on the local system.

func (executor *LocalExecutor) ExecuteShortCommand(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	res, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Failed to create stderr pipe: %s", err)
	}
	return string(res), nil
}

func (executor *LocalExecutor) ExecuteCommand(command string, logChan chan LogEntry) error {
	cmd := exec.Command("sh", "-c", command)
	return executor.executeCommand(cmd, logChan)
}

func (executor *LocalExecutor) executeCommand(cmd *exec.Cmd, logChan chan LogEntry) error {
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stdout for local command: %v", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("Failed to create stderr pipe: %s", err)
	}

	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			text, _ := DecodeGBK(scanner.Bytes())
			logChan <- LogEntry{Message: text, IsError: false}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			text, _ := DecodeGBK(scanner.Bytes())
			logChan <- LogEntry{Message: text, IsError: true}
		}
	}()

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("Failed to run local command: %s", err)
	}

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("Local command execution failed: %s", err)
	}
	return nil
}

// CopyFile copies a file locally.
func (executor *LocalExecutor) CopyFile(srcFile, destFile string, outputHandler func(string)) error {
	src, err := os.Open(srcFile)
	if err != nil {
		return fmt.Errorf("Failed to open source file: %s", err)
	}
	defer src.Close()

	dest, err := os.Create(destFile)
	if err != nil {
		return fmt.Errorf("Failed to create destination file: %s", err)
	}
	defer dest.Close()

	_, err = io.Copy(dest, src)
	if err != nil {
		return fmt.Errorf("Failed to copy file: %s", err)
	}

	outputHandler(fmt.Sprintf("Copied file %s to %s", srcFile, destFile))
	return nil
}
