package nacos

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"go.uber.org/zap"
)

// Client Nacos 客户端封装（配置中心 + 服务注册发现）
type Client struct {
	configClient config_client.IConfigClient
	namingClient naming_client.INamingClient
	namespace    string
	group        string
	logger       *zap.Logger
}

// Config Nacos 客户端配置
type Config struct {
	ServerAddr string // Nacos 服务器地址，如 "192.168.192.87:8848"
	Namespace  string // 命名空间 ID，如 "zebra-dev"
	Username   string // 用户名
	Password   string // 密码
	Group      string // 配置分组，默认 DEFAULT_GROUP
	LogLevel   string // 日志级别: debug, info, warn, error
}

// NewClient 创建 Nacos 客户端
func NewClient(cfg Config, logger *zap.Logger) (*Client, error) {
	if cfg.Group == "" {
		cfg.Group = constant.DEFAULT_GROUP
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	host, port, err := parseServerAddr(cfg.ServerAddr)
	if err != nil {
		return nil, err
	}

	// 构建服务端配置
	serverConfig := []constant.ServerConfig{
		{
			IpAddr: host,
			Port:   port,
		},
	}

	// 客户端配置
	clientConfig := constant.ClientConfig{
		NamespaceId:         cfg.Namespace,
		Username:            cfg.Username,
		Password:            cfg.Password,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogLevel:            cfg.LogLevel,
		LogDir:              "logs/nacos",
		CacheDir:            "cache/nacos",
	}

	// 创建配置客户端
	configClient, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ServerConfigs: serverConfig,
			ClientConfig:  &clientConfig,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create config client: %w", err)
	}

	// 创建服务发现客户端
	namingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ServerConfigs: serverConfig,
			ClientConfig:  &clientConfig,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create naming client: %w", err)
	}

	logger.Info("Nacos 客户端初始化成功",
		zap.String("server", cfg.ServerAddr),
		zap.String("namespace", cfg.Namespace),
	)

	return &Client{
		configClient: configClient,
		namingClient: namingClient,
		namespace:    cfg.Namespace,
		group:        cfg.Group,
		logger:       logger,
	}, nil
}

func parseServerAddr(serverAddr string) (string, uint64, error) {
	trimmed := strings.TrimSpace(serverAddr)
	if trimmed == "" {
		return "", 0, fmt.Errorf("nacos server address is empty")
	}

	host, portText, err := net.SplitHostPort(trimmed)
	if err == nil {
		port, convErr := strconv.ParseUint(portText, 10, 64)
		if convErr != nil {
			return "", 0, fmt.Errorf("parse nacos port %q: %w", portText, convErr)
		}
		return host, port, nil
	}

	if strings.Contains(err.Error(), "missing port in address") {
		return trimmed, 8848, nil
	}

	return "", 0, fmt.Errorf("parse nacos server address %q: %w", serverAddr, err)
}

// GetConfig 获取配置
func (c *Client) GetConfig(dataID string, group string) (string, error) {
	if group == "" {
		group = c.group
	}

	content, err := c.configClient.GetConfig(vo.ConfigParam{
		DataId: dataID,
		Group:  group,
	})
	if err != nil {
		c.logger.Warn("从 Nacos 获取配置失败",
			zap.String("dataID", dataID),
			zap.String("group", group),
			zap.Error(err),
		)
		return "", err
	}

	c.logger.Debug("从 Nacos 获取配置成功",
		zap.String("dataID", dataID),
		zap.Int("length", len(content)),
	)
	return content, nil
}

// ListenConfig 监听配置变更
func (c *Client) ListenConfig(dataID string, group string, onChange func(namespace, group, dataID, data string)) error {
	if group == "" {
		group = c.group
	}

	err := c.configClient.ListenConfig(vo.ConfigParam{
		DataId: dataID,
		Group:  group,
		OnChange: func(namespace, group, dataId, data string) {
			c.logger.Info("配置变更通知",
				zap.String("dataID", dataId),
				zap.String("group", group),
			)
			onChange(namespace, group, dataId, data)
		},
	})

	if err != nil {
		return fmt.Errorf("listen config: %w", err)
	}

	c.logger.Info("添加配置监听器成功", zap.String("dataID", dataID))
	return nil
}

// RegisterInstance 注册服务实例
func (c *Client) RegisterInstance(serviceName, ip string, port uint64, metadata map[string]string) error {
	param := vo.RegisterInstanceParam{
		Ip:          ip,
		Port:        port,
		ServiceName: serviceName,
		GroupName:   c.group,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true, // 临时实例，服务停止自动注销
		Metadata:    metadata,
	}

	var lastErr error
	for attempt := 1; attempt <= 10; attempt++ {
		success, err := c.namingClient.RegisterInstance(param)
		if err == nil && success {
			c.logger.Info("服务注册成功",
				zap.String("service", serviceName),
				zap.String("ip", ip),
				zap.Uint64("port", port),
			)
			return nil
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("register instance returned success=false")
		}

		if attempt < 10 {
			c.logger.Warn("服务注册未就绪，准备重试",
				zap.String("service", serviceName),
				zap.Int("attempt", attempt),
				zap.Error(lastErr),
			)
			time.Sleep(time.Second)
		}
	}

	return fmt.Errorf("register instance after retries: %w", lastErr)
}

// DeregisterInstance 注销服务实例
func (c *Client) DeregisterInstance(serviceName, ip string, port uint64) error {
	success, err := c.namingClient.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          ip,
		Port:        port,
		ServiceName: serviceName,
		GroupName:   c.group,
	})

	if err != nil || !success {
		return fmt.Errorf("deregister instance: success=%v, err=%w", success, err)
	}

	c.logger.Info("服务注销成功",
		zap.String("service", serviceName),
		zap.String("ip", ip),
		zap.Uint64("port", port),
	)
	return nil
}

// SelectOneHealthyInstance 选择一个健康实例（负载均衡）
func (c *Client) SelectOneHealthyInstance(serviceName string) (*Instance, error) {
	instance, err := c.namingClient.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		ServiceName: serviceName,
		GroupName:   c.group,
	})

	if err != nil {
		c.logger.Warn("查询健康实例失败",
			zap.String("service", serviceName),
			zap.Error(err),
		)
		return nil, err
	}

	c.logger.Debug("发现健康实例",
		zap.String("service", serviceName),
		zap.String("ip", instance.Ip),
		zap.Uint64("port", instance.Port),
	)

	return &Instance{
		IP:       instance.Ip,
		Port:     instance.Port,
		Metadata: instance.Metadata,
		Weight:   instance.Weight,
		Healthy:  instance.Healthy,
	}, nil
}

// GetAllInstances 获取服务的所有实例
func (c *Client) GetAllInstances(serviceName string, healthyOnly bool) ([]Instance, error) {
	instances, err := c.namingClient.SelectInstances(vo.SelectInstancesParam{
		ServiceName: serviceName,
		GroupName:   c.group,
		HealthyOnly: healthyOnly,
	})

	if err != nil {
		c.logger.Warn("查询服务实例列表失败",
			zap.String("service", serviceName),
			zap.Error(err),
		)
		return nil, err
	}

	result := make([]Instance, len(instances))
	for i, inst := range instances {
		result[i] = Instance{
			IP:       inst.Ip,
			Port:     inst.Port,
			Metadata: inst.Metadata,
			Weight:   inst.Weight,
			Healthy:  inst.Healthy,
		}
	}

	c.logger.Debug("查询服务实例列表成功",
		zap.String("service", serviceName),
		zap.Int("count", len(result)),
	)

	return result, nil
}

// Instance 服务实例信息
type Instance struct {
	IP       string
	Port     uint64
	Metadata map[string]string
	Weight   float64
	Healthy  bool
}
