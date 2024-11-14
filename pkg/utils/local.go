package utils

import (
	"bufio"
	"fmt"
	"github.com/whoisfisher/mykubespray/pkg/logger"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
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
		log.Printf("Failed to create stderr pipe: %s", err.Error())
		return "", err
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
		log.Printf("Unable to setup stdout for local command: %s", err.Error())
		return err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("Failed to create stderr pipe: %s", err.Error())
		return err
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
		log.Printf("Failed to run local command: %s", err.Error())
		return err
	}

	err = cmd.Wait()
	if err != nil {
		log.Printf("Local command execution failed: %s", err.Error())
		return err
	}
	return nil
}

// CopyFile copies a file locally.
func (executor *LocalExecutor) CopyFile(srcFile, destFile string, outputHandler func(string)) error {
	src, err := os.Open(srcFile)
	if err != nil {
		log.Printf("Failed to open source file: %s", err.Error())
		return err
	}
	defer src.Close()

	dest, err := os.Create(destFile)
	if err != nil {
		log.Printf("Failed to create destination file: %s", err.Error())
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, src)
	if err != nil {
		log.Printf("Failed to copy file: %s", err.Error())
		return err
	}

	outputHandler(fmt.Sprintf("Copied file %s to %s", srcFile, destFile))
	return nil
}

func (executor *LocalExecutor) MkDirALL(path string, outputHandler func(string)) error {
	cmd := exec.Command("/usr/bin/mkdir -p %s", path)
	outputHandler(fmt.Sprintf("Mkdir Directory: %s", path))

	err := cmd.Run()
	if err != nil {
		log.Printf("failed to run command: %s", err.Error())
		return err
	}

	outputHandler(fmt.Sprintf("Mkdir Directory: %s", path))
	return nil
}

func (executor *LocalExecutor) ReadFile(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", path, err)
	}
	return content, nil
}

func (executor *LocalExecutor) WriteFile(content []byte, path string, perm os.FileMode) error {
	err := os.WriteFile(path, content, perm)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %v", path, err)
	}
	return nil
}

func (executor *LocalExecutor) WhoAmI() string {
	command := fmt.Sprintf("whoami")
	user, err := executor.ExecuteShortCommand(command)
	if err != nil {
		logger.GetLogger().Warnf("Read username failed: %v", err.Error())
		return ""
	}
	return strings.TrimSpace(user)
}
