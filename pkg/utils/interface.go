package utils

import "os"

// Executor interface defines methods for executing commands and copying files.
type Executor interface {
	ExecuteCommand(command string, logChan chan LogEntry) error
	CopyFile(srcFile, destFile string, outputHandler func(string)) error
	CopyRemoteToRemote(srcHost, srcFile, destHost, destFile string, outputHandler func(string)) error
	ReadFile(path string) ([]byte, error)
	WriteFile(content []byte, path string, perm os.FileMode)
	MkDirALL(path string, outputHandler func(string)) error
	ExecuteShortCommand(command string) (string, error)
}

// Connection interface defines methods for establishing a connection.
type Connection interface {
	Connect(config SSHConfig) error
}
