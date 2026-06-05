package ssh

import (
	"bytes"
	"fmt"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SSHClient struct {
	Host    string
	Port    int
	User    string
	Timeout time.Duration
	client  *ssh.Client
}

// NewSSHClient 创建仅使用密钥认证的SSH客户端
func NewSSHClient(host string, port int, user string, signer ssh.Signer) (*SSHClient, error) {
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	c, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}
	return &SSHClient{Host: host, Port: port, User: user, client: c}, nil
}

// NewSSHClientWithAuth 创建支持多种认证方式的SSH客户端（密码/密钥/两者兼有）
func NewSSHClientWithAuth(host string, port int, user string, authMethods []ssh.AuthMethod) (*SSHClient, error) {
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	c, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}
	return &SSHClient{Host: host, Port: port, User: user, client: c}, nil
}

// NewSSHClientWithTimeout 创建带自定义超时配置的SSH客户端
func NewSSHClientWithTimeout(host string, port int, user string, authMethods []ssh.AuthMethod, timeout time.Duration) (*SSHClient, error) {
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	c, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}
	return &SSHClient{Host: host, Port: port, User: user, Timeout: timeout, client: c}, nil
}

// Close 释放SSH连接
func (s *SSHClient) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// Run 执行命令并返回stdout和stderr
func (s *SSHClient) Run(cmd string) (string, string, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return "", "", err
	}
	defer session.Close()
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	session.Stdout = &outBuf
	session.Stderr = &errBuf
	if err := session.Run(cmd); err != nil {
		return outBuf.String(), errBuf.String(), err
	}
	return outBuf.String(), errBuf.String(), nil
}

// RunCommandOutput 执行命令并返回stdout、stderr和退出码
func (s *SSHClient) RunCommandOutput(cmd string) (string, string, int, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return "", "", -1, err
	}
	defer session.Close()
	var outBuf, errBuf bytes.Buffer
	session.Stdout = &outBuf
	session.Stderr = &errBuf
	err = session.Run(cmd)
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
		} else {
			exitCode = -1
		}
	}
	return outBuf.String(), errBuf.String(), exitCode, err
}

// UploadFile 通过SFTP上传文件到远程主机
func (s *SSHClient) UploadFile(remotePath string, content []byte) error {
	sftpClient, err := sftp.NewClient(s.client)
	if err != nil {
		return fmt.Errorf("create sftp client: %w", err)
	}
	defer sftpClient.Close()

	dstFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("create remote file %s: %w", remotePath, err)
	}
	defer dstFile.Close()

	_, err = dstFile.Write(content)
	if err != nil {
		return fmt.Errorf("write remote file %s: %w", remotePath, err)
	}
	return nil
}