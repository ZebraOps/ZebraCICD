package nacos

import (
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// CICDConfig CICD 服务的 YAML 配置结构
type CICDConfig struct {
	Database struct {
		URL            string `yaml:"url"`
		ConnectionPool struct {
			MaxIdleConns int `yaml:"max_idle_conns"`
			MaxOpenConns int `yaml:"max_open_conns"`
		} `yaml:"connection_pool"`
	} `yaml:"database"`
	
	GitLab struct {
		URL     string `yaml:"url"`
		Token   string `yaml:"token"`
		Timeout string `yaml:"timeout"`
	} `yaml:"gitlab"`
	
	Jenkins struct {
		URL      string `yaml:"url"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Timeout  string `yaml:"timeout"`
	} `yaml:"jenkins"`
	
	Registry struct {
		URL      string `yaml:"url"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"registry"`
	
	Kubernetes struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"kubernetes"`
	
	WebSocket struct {
		Enabled        bool `yaml:"enabled"`
		MaxConnections int  `yaml:"max_connections"`
	} `yaml:"websocket"`

	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`

	Worker struct {
		Concurrency int `yaml:"concurrency"`
	} `yaml:"worker"`

	Service struct {
		Name        string `yaml:"name"`
		Version     string `yaml:"version"`
		Description string `yaml:"description"`
	} `yaml:"service"`
}

// CICDConfigLoader CICD 专用配置加载器
type CICDConfigLoader struct {
	client *Client
	logger *zap.Logger
	config *CICDConfig
}

// NewCICDConfigLoader 创建 CICD 配置加载器
func NewCICDConfigLoader(client *Client, logger *zap.Logger) *CICDConfigLoader {
	loader := &CICDConfigLoader{
		client: client,
		logger: logger,
	}
	
	// 启动时加载 YAML 配置
	if err := loader.loadYAMLConfig(); err != nil {
		logger.Error("加载 Nacos YAML 配置失败", zap.Error(err))
	}
	
	return loader
}

// loadYAMLConfig 从 Nacos 加载 YAML 配置文件
func (l *CICDConfigLoader) loadYAMLConfig() error {
	content, err := l.client.GetConfig("zebra-cicd.yaml", "DEFAULT_GROUP")
	if err != nil {
		l.logger.Error("获取 YAML 配置失败", zap.Error(err))
		return err
	}
	
	if content == "" {
		l.logger.Warn("Nacos 中未找到 zebra-cicd.yaml 配置")
		return nil
	}
	
	var config CICDConfig
	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		l.logger.Error("解析 YAML 配置失败", zap.Error(err))
		return err
	}
	
	l.config = &config
	l.logger.Info("✓ 从 Nacos 加载 YAML 配置成功")
	return nil
}

// LoadDatabaseURL 加载数据库连接字符串
func (l *CICDConfigLoader) LoadDatabaseURL(defaultValue string) string {
	if l.config != nil && l.config.Database.URL != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载数据库配置成功")
		return l.config.Database.URL
	}
	l.logger.Warn("从 Nacos 加载数据库配置失败，使用默认值")
	return defaultValue
}

// LoadGitLabToken 加载 GitLab Token
func (l *CICDConfigLoader) LoadGitLabToken(defaultValue string) string {
	if l.config != nil && l.config.GitLab.Token != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 GitLab Token 成功")
		return l.config.GitLab.Token
	}
	l.logger.Warn("从 Nacos 加载 GitLab Token 失败，使用默认值")
	return defaultValue
}

// LoadGitLabURL 加载 GitLab URL
func (l *CICDConfigLoader) LoadGitLabURL(defaultValue string) string {
	if l.config != nil && l.config.GitLab.URL != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 GitLab URL 成功")
		return l.config.GitLab.URL
	}
	return defaultValue
}

// LoadJenkinsURL 加载 Jenkins URL
func (l *CICDConfigLoader) LoadJenkinsURL(defaultValue string) string {
	if l.config != nil && l.config.Jenkins.URL != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 Jenkins URL 成功")
		return l.config.Jenkins.URL
	}
	return defaultValue
}

// LoadJenkinsPassword 加载 Jenkins 密码
func (l *CICDConfigLoader) LoadJenkinsPassword(defaultValue string) string {
	if l.config != nil && l.config.Jenkins.Password != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 Jenkins 密码成功")
		return l.config.Jenkins.Password
	}
	l.logger.Warn("从 Nacos 加载 Jenkins 密码失败，使用默认值")
	return defaultValue
}

// LoadRegistryURL 加载 Registry URL
func (l *CICDConfigLoader) LoadRegistryURL(defaultValue string) string {
	if l.config != nil && l.config.Registry.URL != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 Registry URL 成功")
		return l.config.Registry.URL
	}
	return defaultValue
}

// LoadRedisAddr 加载 Redis 地址
func (l *CICDConfigLoader) LoadRedisAddr(defaultValue string) string {
	if l.config != nil && l.config.Redis.Addr != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 Redis 地址成功")
		return l.config.Redis.Addr
	}
	return defaultValue
}

// LoadRedisPassword 加载 Redis 密码
func (l *CICDConfigLoader) LoadRedisPassword(defaultValue string) string {
	if l.config != nil {
		return l.config.Redis.Password
	}
	return defaultValue
}

// LoadAllConfigs 一次性加载所有配置
func (l *CICDConfigLoader) LoadAllConfigs(cfg map[string]string) {
	configs := []struct {
		key          string
		defaultValue string
		loader       func(string) string
	}{
		{"database_url", cfg["database_url"], l.LoadDatabaseURL},
		{"gitlab_token", cfg["gitlab_token"], l.LoadGitLabToken},
		{"gitlab_url", cfg["gitlab_url"], l.LoadGitLabURL},
		{"jenkins_url", cfg["jenkins_url"], l.LoadJenkinsURL},
		{"jenkins_password", cfg["jenkins_password"], l.LoadJenkinsPassword},
		{"registry_url", cfg["registry_url"], l.LoadRegistryURL},
		{"redis_addr", cfg["redis_addr"], l.LoadRedisAddr},
		{"redis_password", cfg["redis_password"], l.LoadRedisPassword},
	}

	for _, c := range configs {
		cfg[c.key] = c.loader(c.defaultValue)
	}
}
