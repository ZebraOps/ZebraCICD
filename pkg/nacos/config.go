package nacos

import (
	"fmt"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// ConfigLoader Nacos 配置加载器
type ConfigLoader struct {
	client *Client
	logger *zap.Logger
}

// NewConfigLoader 创建配置加载器
func NewConfigLoader(client *Client, logger *zap.Logger) *ConfigLoader {
	return &ConfigLoader{
		client: client,
		logger: logger,
	}
}

// LoadDatabaseURL 加载数据库连接字符串
func (l *ConfigLoader) LoadDatabaseURL(defaultValue string) string {
	content, err := l.client.GetConfig("zebra.gateway.database.url", "")
	if err != nil || content == "" {
		l.logger.Warn("从 Nacos 加载数据库配置失败，使用默认值", zap.Error(err))
		return defaultValue
	}
	l.logger.Info("从 Nacos 加载数据库配置成功")
	return content
}

// LoadJWTSecret 加载 JWT 密钥
func (l *ConfigLoader) LoadJWTSecret(defaultValue string) string {
	content, err := l.client.GetConfig("zebra.gateway.jwt-secret", "")
	if err != nil || content == "" {
		l.logger.Warn("从 Nacos 加载 JWT 密钥失败，使用默认值", zap.Error(err))
		return defaultValue
	}
	l.logger.Info("从 Nacos 加载 JWT 密钥成功")
	return content
}

// LoadCacheTTL 加载缓存 TTL（秒）
func (l *ConfigLoader) LoadCacheTTL(defaultValue int) int {
	content, err := l.client.GetConfig("zebra.gateway.cache-ttl", "")
	if err != nil || content == "" {
		return defaultValue
	}

	ttl, err := strconv.Atoi(content)
	if err != nil {
		l.logger.Warn("解析 CacheTTL 失败", zap.Error(err))
		return defaultValue
	}

	l.logger.Info("从 Nacos 加载 CacheTTL 成功", zap.Int("ttl", ttl))
	return ttl
}

// LoadRouteReloadInterval 加载路由重载间隔
func (l *ConfigLoader) LoadRouteReloadInterval(defaultValue time.Duration) time.Duration {
	content, err := l.client.GetConfig("zebra.gateway.route-reload-interval", "")
	if err != nil || content == "" {
		return defaultValue
	}

	interval, err := time.ParseDuration(content)
	if err != nil {
		l.logger.Warn("解析 RouteReloadInterval 失败", zap.Error(err))
		return defaultValue
	}

	l.logger.Info("从 Nacos 加载 RouteReloadInterval 成功", zap.Duration("interval", interval))
	return interval
}

// DiscoverRBACService 通过服务发现获取 ZebraRBAC 服务地址
// 返回格式: http://ip:port
func (l *ConfigLoader) DiscoverRBACService() (string, error) {
	instance, err := l.client.SelectOneHealthyInstance("zebra-rbac")
	if err != nil {
		return "", fmt.Errorf("discover zebra-rbac: %w", err)
	}

	if instance == nil {
		return "", fmt.Errorf("zebra-rbac service not found")
	}

	url := fmt.Sprintf("http://%s:%d", instance.IP, instance.Port)
	l.logger.Info("通过服务发现获取 ZebraRBAC 地址", zap.String("url", url))
	return url, nil
}

// WatchCacheTTL 监听 CacheTTL 配置变更
func (l *ConfigLoader) WatchCacheTTL(onChange func(int)) error {
	return l.client.ListenConfig("zebra.gateway.cache-ttl", "", func(namespace, group, dataID, data string) {
		ttl, err := strconv.Atoi(data)
		if err != nil {
			l.logger.Error("解析新的 CacheTTL 失败", zap.Error(err))
			return
		}
		l.logger.Info("CacheTTL 配置已变更", zap.Int("new_ttl", ttl))
		onChange(ttl)
	})
}

// WatchRouteReloadInterval 监听路由重载间隔配置变更
func (l *ConfigLoader) WatchRouteReloadInterval(onChange func(time.Duration)) error {
	return l.client.ListenConfig("zebra.gateway.route-reload-interval", "", func(namespace, group, dataID, data string) {
		interval, err := time.ParseDuration(data)
		if err != nil {
			l.logger.Error("解析新的 RouteReloadInterval 失败", zap.Error(err))
			return
		}
		l.logger.Info("RouteReloadInterval 配置已变更", zap.Duration("new_interval", interval))
		onChange(interval)
	})
}
