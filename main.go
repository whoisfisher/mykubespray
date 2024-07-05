package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
	"os/exec"
)

// SSHConfig 结构体用于配置 SSH 连接信息
type SSHConfig struct {
	Host        string           // 主机地址
	Port        int              // 端口号
	User        string           // 用户名
	PrivateKey  string           // 私钥文件路径（用于公钥认证）
	Password    string           // 密码（用于密码认证）
	AuthMethods []ssh.AuthMethod // 认证方法
}

// SSHConnection 结构体用于保存 SSH 连接的客户端
type SSHConnection struct {
	Client *ssh.Client
}

// SSHSession 结构体用于保存 SSH 会话
type SSHSession struct {
	Session *ssh.Session
}

// LocalExecutor 结构体用于本地命令执行
type LocalExecutor struct{}

// connectSSH 函数用于连接到 SSH 主机
func connectSSH(config SSHConfig) (*SSHConnection, error) {
	sshConfig := &ssh.ClientConfig{
		User:            config.User,
		Auth:            config.AuthMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	address := fmt.Sprintf("%s:%d", config.Host, config.Port)
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to dial: %s", err)
	}

	connection := &SSHConnection{
		Client: client,
	}

	return connection, nil
}

// createSession 方法用于创建 SSH 会话
func (conn *SSHConnection) createSession() (*SSHSession, error) {
	session, err := conn.Client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("Failed to create session: %s", err)
	}

	sshSession := &SSHSession{
		Session: session,
	}

	return sshSession, nil
}

// executeRemoteCommand 方法用于执行远程命令
func (sess *SSHSession) executeRemoteCommand(command string, outputHandler func(string)) error {
	stdoutPipe, err := sess.Session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stdout for session: %v", err)
	}

	output := make(chan string)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdoutPipe.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Fatalf("Error reading from stdout: %v", err)
			}
			output <- string(buf[:n])
		}
		close(output)
	}()

	err = sess.Session.Start(command)
	if err != nil {
		return fmt.Errorf("Failed to run command: %s", err)
	}

	for line := range output {
		outputHandler(line)
	}

	err = sess.Session.Wait()
	if err != nil {
		return fmt.Errorf("Command execution failed: %s", err)
	}

	return nil
}

// executeLocalCommand 方法用于执行本地命令
func (executor LocalExecutor) executeLocalCommand(command string, outputHandler func(string)) error {
	cmd := exec.Command("sh", "-c", command)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stdout for local command: %v", err)
	}

	output := make(chan string)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdoutPipe.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Fatalf("Error reading from stdout: %v", err)
			}
			output <- string(buf[:n])
		}
		close(output)
	}()

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("Failed to run local command: %s", err)
	}

	for line := range output {
		outputHandler(line)
	}

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("Local command execution failed: %s", err)
	}

	return nil
}

// scpLocalToRemote 方法用于将本地文件复制到远程主机
func (sess *SSHSession) scpLocalToRemote(srcFile, destFile string, outputHandler func(string)) error {
	file, err := os.Open(srcFile)
	if err != nil {
		return fmt.Errorf("Failed to open local file: %s", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("Failed to get file stat: %s", err)
	}

	session := sess.Session
	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
		fmt.Fprintf(w, "C0644 %d %s\n", stat.Size(), destFile)
		io.Copy(w, file)
		fmt.Fprint(w, "\x00")
	}()

	err = session.Start("/usr/bin/scp -t " + destFile)
	if err != nil {
		return fmt.Errorf("Failed to start SCP session: %s", err)
	}

	err = session.Wait()
	if err != nil {
		return fmt.Errorf("SCP execution failed: %s", err)
	}

	return nil
}

func main() {
	// 示例用法

	// 定义 SSH 连接配置
	sshConfig := SSHConfig{
		Host: "remote-host",
		Port: 2222, // 修改为实际的 SSH 端口号
		User: "username",
		// 如果使用私钥认证，设置 PrivateKey 字段，例如：
		// PrivateKey: "/path/to/private_key",
		// 如果使用密码认证，设置 Password 字段，例如：
		Password: "password",
		// 可以添加其他认证方法，例如：
		// AuthMethods: []ssh.AuthMethod{
		//     ssh.Password("password"),
		//     ssh.PublicKeys(publicKey),
		// },
	}

	// 建立 SSH 连接
	conn, err := connectSSH(sshConfig)
	if err != nil {
		log.Fatalf("Error connecting to SSH: %s", err)
	}
	defer conn.Client.Close()

	// 创建 SSH 会话
	session, err := conn.createSession()
	if err != nil {
		log.Fatalf("Error creating SSH session: %s", err)
	}
	defer session.Session.Close()

	// 使用 SCP 将本地文件复制到远程主机
	err = session.scpLocalToRemote("/path/to/localfile", "/path/to/remotefile", func(line string) {
		fmt.Println(line) // 输出到本地控制台
	})
	if err != nil {
		log.Fatalf("Error copying file from local to remote: %s", err)
	}

	// 示例：执行远程命令
	err = session.executeRemoteCommand("ls -l", func(line string) {
		fmt.Println(line) // 输出到本地控制台
	})
	if err != nil {
		log.Fatalf("Error running remote command: %s", err)
	}
}
