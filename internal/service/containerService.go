package service

import (
	"fmt"
	"io"

	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/ZebraOps/ZebraCICD/pkg/log"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

// ExecContainer 在容器中执行命令
func (s *ServerService) ExecContainer(serverID uint, containerID, command string) (*types.ContainerExecResponse, error) {
	server, err := s.serverRepo.GetByID(serverID)
	if err != nil {
		return nil, err
	}

	sshClient, err := s.createSSHClient(server)
	if err != nil {
		return nil, err
	}
	defer sshClient.Close()

	// Detect DOCKER_HOST compatibility (mirrors deployToDocker fallback logic)
	dockerEnv := s.detectDockerEnv(sshClient)

	session, err := sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	// 执行docker exec命令
	fullCmd := fmt.Sprintf(dockerEnv+"docker exec %s %s", containerID, command)
	output, err := session.CombinedOutput(fullCmd)

	return &types.ContainerExecResponse{
		Output: string(output),
		Error:  err,
	}, nil
}

// wsWriter wraps a WebSocket connection as an io.Writer for use with io.Copy.
type wsWriter struct {
	conn *websocket.Conn
}

func (w *wsWriter) Write(p []byte) (int, error) {
	if err := w.conn.WriteMessage(websocket.TextMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

// AttachContainer 连接到容器，通过 WebSocket 提供交互式终端（类似 docker exec -it）。
//
// 流程：
//  1. SSH 连接到目标服务器
//  2. 分配 PTY（伪终端）以支持交互式 shell
//  3. 执行 docker exec -it <container> /bin/sh
//  4. 双向桥接：stdout → WebSocket，WebSocket → stdin
func (s *ServerService) AttachContainer(serverID uint, containerID string, wsConn *websocket.Conn) error {
	server, err := s.serverRepo.GetByID(serverID)
	if err != nil {
		return err
	}

	sshClient, err := s.createSSHClient(server)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	// Detect DOCKER_HOST compatibility (mirrors deployToDocker fallback logic)
	dockerEnv := s.detectDockerEnv(sshClient)

	session, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// 分配 PTY 以支持交互式终端（光标、回显等）
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // 启用回显
		ssh.TTY_OP_ISPEED: 14400, // 输入速率
		ssh.TTY_OP_OSPEED: 14400, // 输出速率
	}
	if err := session.RequestPty("xterm", 80, 24, modes); err != nil {
		log.S().Warnf("request pty failed (non-fatal): %v", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}

	// 使用 docker exec -it 启动交互式 shell（不是 docker attach！）
	// docker attach 连接到容器主进程（如 nginx），无法输入指令
	// docker exec -it 创建新的 shell 会话
	attachCmd := dockerEnv + "docker exec -it " + containerID + " /bin/sh"
	if err := session.Start(attachCmd); err != nil {
		log.S().Errorf("start docker exec failed: %v", err)
		return err
	}

	// stdout → WebSocket（使用 io.Copy 而非 bufio.Scanner，避免 shell 提示符丢失）
	go func() {
		if _, err := io.Copy(&wsWriter{conn: wsConn}, stdout); err != nil {
			log.S().Debugf("stdout→ws copy ended: %v", err)
		}
	}()

	// WebSocket → stdin
	for {
		_, message, err := wsConn.ReadMessage()
		if err != nil {
			log.S().Debugf("websocket read closed: %v", err)
			break
		}

		if _, err := stdin.Write(message); err != nil {
			log.S().Errorf("write to stdin failed: %v", err)
			break
		}
	}

	return nil
}
