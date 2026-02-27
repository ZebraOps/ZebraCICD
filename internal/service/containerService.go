package service

import (
	"bufio"
	"fmt"

	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/ZebraOps/ZebraCICD/pkg/log"
	"github.com/gorilla/websocket" // 确保添加 websocket 导入
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

	session, err := sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	// 执行docker exec命令
	fullCmd := fmt.Sprintf("docker exec %s %s", containerID, command)
	output, err := session.CombinedOutput(fullCmd)

	return &types.ContainerExecResponse{
		Output: string(output),
		Error:  err,
	}, nil
}

// AttachContainer 连接到容器
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

	session, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// 设置标准输入输出
	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}

	// 启动docker attach命令
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if err := wsConn.WriteMessage(websocket.TextMessage, []byte(scanner.Text()+"\n")); err != nil {
				log.S().Errorf("write to websocket failed: %v", err)
				break
			}
		}
	}()

	// 处理来自WebSocket的消息
	for {
		_, message, err := wsConn.ReadMessage()
		if err != nil {
			log.S().Infof("websocket closed: %v", err)
			break
		}

		if _, err := stdin.Write(message); err != nil {
			log.S().Errorf("write to stdin failed: %v", err)
			break
		}
	}

	return nil
}
