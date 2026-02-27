package ssh

import (
	"bytes"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHClient struct {
	Host    string
	Port    int
	User    string
	Timeout time.Duration
	client  *ssh.Client
}

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
