package service

import (
	"encoding/json"
	"fmt"
	"strings" // 添加缺失的导入
	"time"    // 添加缺失的导入

	"github.com/ZebraOps/ZebraCICD/internal/handler"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/ZebraOps/ZebraCICD/pkg/timeutil"
	"golang.org/x/crypto/ssh"
)

type ServerService struct {
	serverRepo *handler.ServerRepository
}

func NewServerService(serverRepo *handler.ServerRepository) *ServerService {
	return &ServerService{
		serverRepo: serverRepo,
	}
}

// CreateServer 创建服务器连接信息
func (s *ServerService) CreateServer(server *model.Server) error {
	return s.serverRepo.Create(server)
}

// GetServerByID 根据ID获取服务器
func (s *ServerService) GetServerByID(id uint) (*model.Server, error) {
	return s.serverRepo.GetByID(id)
}

// ListServersWithConditions 根据条件分页获取服务器列表
func (s *ServerService) ListServersWithConditions(conditions types.ServerQueryConditions, page, size int) ([]model.Server, int64, error) {
	return s.serverRepo.ListWithConditions(conditions, page, size)
}

// UpdateServer 更新服务器信息
func (s *ServerService) UpdateServer(server *model.Server) error {
	return s.serverRepo.Update(server)
}

// DeleteServer 删除服务器
func (s *ServerService) DeleteServer(id uint) error {
	return s.serverRepo.Delete(id)
}

// TestConnection 测试连接服务器
func (s *ServerService) TestConnection(serverID uint) error {
	server, err := s.serverRepo.GetByID(serverID)
	if err != nil {
		return err
	}

	// 创建SSH客户端
	sshClient, err := s.createSSHClient(server)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	// 执行简单命令测试连接
	session, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	_, err = session.Output("echo 'connection test'")
	return err
}

// ListContainers 获取Docker容器列表
func (s *ServerService) ListContainers(serverID uint) ([]model.DockerContainer, error) {
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

	// 执行docker ps命令获取容器列表
	output, err := session.Output("docker ps --format '{{.ID}}\\t{{.Names}}\\t{{.Image}}\\t{{.Command}}\\t{{.Status}}\\t{{.Ports}}\\t{{.Labels}}'")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	var containers []model.DockerContainer

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) >= 7 {
			// 处理 Names 字段 - 直接使用原始名称，不转换为JSON
			names := strings.Split(parts[1], ",")
			// 清理名称数组中的空格
			cleanedNames := make([]string, 0)
			for _, name := range names {
				trimmedName := strings.TrimSpace(name)
				if trimmedName != "" {
					cleanedNames = append(cleanedNames, trimmedName)
				}
			}

			// 处理 Command 字段 - 移除引号和省略号
			command := strings.TrimPrefix(parts[3], "\"")
			command = strings.TrimSuffix(command, "\"")
			if strings.HasSuffix(command, "…") {
				command = strings.TrimSuffix(command, "…")
			}

			// 创建容器对象
			container := model.DockerContainer{
				ID:      parts[0],
				Names:   cleanedNames, // 直接存储名称数组
				Image:   parts[2],
				Command: command, // 清理后的命令
				Status:  parts[4],
				Ports:   parts[5],
			}

			// 处理 Labels 字段
			labels := parseLabels(parts[6])
			labelsJSON, _ := json.Marshal(labels)
			container.Labels = string(labelsJSON)

			// 获取容器创建时间
			creationTime, err := s.getContainerCreationTime(sshClient, container.ID)
			if err == nil {
				container.CreatedAt = timeutil.JSONTime(creationTime)
			}

			containers = append(containers, container)
		}
	}

	return containers, nil
}

// createSSHClient 创建SSH客户端
func (s *ServerService) createSSHClient(server *model.Server) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	if server.AuthType == "key" {
		// 使用私钥认证
		signer, err := ssh.ParsePrivateKey([]byte(server.PrivateKey))
		if err != nil {
			return nil, err
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else {
		// 使用密码认证
		authMethods = append(authMethods, ssh.Password(server.Password))
	}

	config := &ssh.ClientConfig{
		User:            server.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", server.Host, server.Port)
	return ssh.Dial("tcp", addr, config)
}

// getContainerCreationTime 获取容器创建时间
func (s *ServerService) getContainerCreationTime(client *ssh.Client, containerID string) (time.Time, error) {
	session, err := client.NewSession()
	if err != nil {
		return time.Time{}, err
	}
	defer session.Close()

	output, err := session.Output(fmt.Sprintf("docker inspect --format='{{.Created}}' %s", containerID))
	if err != nil {
		return time.Time{}, err
	}

	return time.Parse(time.RFC3339Nano, strings.TrimSpace(string(output)))
}

// parseLabels 解析Docker标签
func parseLabels(labelsStr string) map[string]string {
	labels := make(map[string]string)
	if labelsStr == "" {
		return labels
	}

	pairs := strings.Split(labelsStr, ",")
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) == 2 {
			labels[kv[0]] = kv[1]
		}
	}

	return labels
}
