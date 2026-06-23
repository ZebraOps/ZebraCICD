package nacos

import (
	"fmt"

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
		URL               string `yaml:"url"`
		Username          string `yaml:"username"`
		Password          string `yaml:"password"`
		Timeout           string `yaml:"timeout"`
		BuildWaitTimeout  string `yaml:"build_wait_timeout"`
		BuildPollInterval string `yaml:"build_poll_interval"`
		DefaultCredentials struct {
			Registry string `yaml:"registry"`
			Git      string `yaml:"git"`
		} `yaml:"default_credentials"`
	} `yaml:"jenkins"`

	Registry struct {
		URL      string `yaml:"default_url"`
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

	// 超时配置
	Timeout struct {
		GitlabHTTP   string `yaml:"gitlab_http"`
		JenkinsHTTP  string `yaml:"jenkins_http"`
		SSHConnect   string `yaml:"ssh_connect"`
		GitTest      string `yaml:"git_test"`
		ServerTest   string `yaml:"server_test"`
		RegistryHTTP string `yaml:"registry_http"`
	} `yaml:"timeout"`

	// 路径配置
	Paths struct {
		DeployBase string `yaml:"deploy_base"`
		NginxConf  string `yaml:"nginx_conf"`
		Secrets    string `yaml:"secrets"`
		Logs       string `yaml:"logs"`
	} `yaml:"paths"`

	// 队列配置
	Queue struct {
		MaxRetry       int    `yaml:"max_retry"`
		TaskTimeout    string `yaml:"task_timeout"`
		RetryDelayBase string `yaml:"retry_delay_base"`
	} `yaml:"queue"`
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

// LoadJenkinsBuildWaitTimeout 加载 Jenkins 构建等待超时
func (l *CICDConfigLoader) LoadJenkinsBuildWaitTimeout(defaultValue string) string {
	if l.config != nil && l.config.Jenkins.BuildWaitTimeout != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 Jenkins BuildWaitTimeout 成功")
		return l.config.Jenkins.BuildWaitTimeout
	}
	return defaultValue
}

// LoadJenkinsBuildPollInterval 加载 Jenkins 构建轮询间隔
func (l *CICDConfigLoader) LoadJenkinsBuildPollInterval(defaultValue string) string {
	if l.config != nil && l.config.Jenkins.BuildPollInterval != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 Jenkins BuildPollInterval 成功")
		return l.config.Jenkins.BuildPollInterval
	}
	return defaultValue
}

// LoadJenkinsDefaultRegistryCred 加载 Jenkins 默认镜像仓库凭据 ID
func (l *CICDConfigLoader) LoadJenkinsDefaultRegistryCred(defaultValue string) string {
	if l.config != nil && l.config.Jenkins.DefaultCredentials.Registry != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 Jenkins DefaultRegistryCred 成功")
		return l.config.Jenkins.DefaultCredentials.Registry
	}
	return defaultValue
}

// LoadJenkinsDefaultGitCred 加载 Jenkins 默认 Git 凭据 ID
func (l *CICDConfigLoader) LoadJenkinsDefaultGitCred(defaultValue string) string {
	if l.config != nil && l.config.Jenkins.DefaultCredentials.Git != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 Jenkins DefaultGitCred 成功")
		return l.config.Jenkins.DefaultCredentials.Git
	}
	return defaultValue
}

// LoadSSHConnectTimeout 加载 SSH 连接超时
func (l *CICDConfigLoader) LoadSSHConnectTimeout(defaultValue string) string {
	if l.config != nil && l.config.Timeout.SSHConnect != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 SSH Connect Timeout 成功")
		return l.config.Timeout.SSHConnect
	}
	return defaultValue
}

// LoadGitlabHTTPTimeout 加载 GitLab HTTP 超时
func (l *CICDConfigLoader) LoadGitlabHTTPTimeout(defaultValue string) string {
	if l.config != nil && l.config.Timeout.GitlabHTTP != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 GitLab HTTP Timeout 成功")
		return l.config.Timeout.GitlabHTTP
	}
	return defaultValue
}

// LoadJenkinsHTTPTimeout 加载 Jenkins HTTP 超时
func (l *CICDConfigLoader) LoadJenkinsHTTPTimeout(defaultValue string) string {
	if l.config != nil && l.config.Timeout.JenkinsHTTP != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 Jenkins HTTP Timeout 成功")
		return l.config.Timeout.JenkinsHTTP
	}
	return defaultValue
}

// LoadDeployBasePath 加载部署基础路径
func (l *CICDConfigLoader) LoadDeployBasePath(defaultValue string) string {
	if l.config != nil && l.config.Paths.DeployBase != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 Deploy Base Path 成功")
		return l.config.Paths.DeployBase
	}
	return defaultValue
}

// LoadNginxConfPath 加载 Nginx 配置路径
func (l *CICDConfigLoader) LoadNginxConfPath(defaultValue string) string {
	if l.config != nil && l.config.Paths.NginxConf != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 Nginx Conf Path 成功")
		return l.config.Paths.NginxConf
	}
	return defaultValue
}

// LoadJenkinsUser 加载 Jenkins 用户名
func (l *CICDConfigLoader) LoadJenkinsUser(defaultValue string) string {
	if l.config != nil && l.config.Jenkins.Username != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 Jenkins 用户名成功")
		return l.config.Jenkins.Username
	}
	return defaultValue
}

// LoadRegistryUsername 加载镜像仓库用户名
func (l *CICDConfigLoader) LoadRegistryUsername(defaultValue string) string {
	if l.config != nil && l.config.Registry.Username != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 Registry 用户名成功")
		return l.config.Registry.Username
	}
	return defaultValue
}

// LoadRegistryPassword 加载镜像仓库密码
func (l *CICDConfigLoader) LoadRegistryPassword(defaultValue string) string {
	if l.config != nil && l.config.Registry.Password != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 Registry 密码成功")
		return l.config.Registry.Password
	}
	return defaultValue
}

// LoadQueueMaxRetry 加载队列最大重试次数
func (l *CICDConfigLoader) LoadQueueMaxRetry(defaultValue string) string {
	if l.config != nil {
		l.logger.Info("✓ 从 Nacos YAML 加载 Queue MaxRetry 成功", zap.Int("value", l.config.Queue.MaxRetry))
		return fmt.Sprintf("%d", l.config.Queue.MaxRetry)
	}
	return defaultValue
}

// LoadQueueTaskTimeout 加载队列任务超时
func (l *CICDConfigLoader) LoadQueueTaskTimeout(defaultValue string) string {
	if l.config != nil && l.config.Queue.TaskTimeout != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 Queue TaskTimeout 成功")
		return l.config.Queue.TaskTimeout
	}
	return defaultValue
}

// LoadQueueRetryDelayBase 加载队列重试延迟基数
func (l *CICDConfigLoader) LoadQueueRetryDelayBase(defaultValue string) string {
	if l.config != nil && l.config.Queue.RetryDelayBase != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 Queue RetryDelayBase 成功")
		return l.config.Queue.RetryDelayBase
	}
	return defaultValue
}

// LoadServiceName 加载服务名称
func (l *CICDConfigLoader) LoadServiceName(defaultValue string) string {
	if l.config != nil && l.config.Service.Name != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 Service Name 成功")
		return l.config.Service.Name
	}
	return defaultValue
}

// LoadServiceDescription 加载服务描述
func (l *CICDConfigLoader) LoadServiceDescription(defaultValue string) string {
	if l.config != nil && l.config.Service.Description != "" {
		l.logger.Info("✓ 从 Nacos YAML 加载 Service Description 成功")
		return l.config.Service.Description
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
		{"jenkins_user", cfg["jenkins_user"], l.LoadJenkinsUser},
		{"jenkins_password", cfg["jenkins_password"], l.LoadJenkinsPassword},
		{"registry_url", cfg["registry_url"], l.LoadRegistryURL},
		{"registry_username", cfg["registry_username"], l.LoadRegistryUsername},
		{"registry_password", cfg["registry_password"], l.LoadRegistryPassword},
		{"redis_addr", cfg["redis_addr"], l.LoadRedisAddr},
		{"redis_password", cfg["redis_password"], l.LoadRedisPassword},
		// Jenkins 详细配置
		{"jenkins_build_wait_timeout", cfg["jenkins_build_wait_timeout"], l.LoadJenkinsBuildWaitTimeout},
		{"jenkins_build_poll_interval", cfg["jenkins_build_poll_interval"], l.LoadJenkinsBuildPollInterval},
		{"jenkins_default_registry_cred", cfg["jenkins_default_registry_cred"], l.LoadJenkinsDefaultRegistryCred},
		{"jenkins_default_git_cred", cfg["jenkins_default_git_cred"], l.LoadJenkinsDefaultGitCred},
		// 超时配置
		{"ssh_connect_timeout", cfg["ssh_connect_timeout"], l.LoadSSHConnectTimeout},
		{"gitlab_http_timeout", cfg["gitlab_http_timeout"], l.LoadGitlabHTTPTimeout},
		{"jenkins_http_timeout", cfg["jenkins_http_timeout"], l.LoadJenkinsHTTPTimeout},
		// 路径配置
		{"deploy_base_path", cfg["deploy_base_path"], l.LoadDeployBasePath},
		{"nginx_conf_path", cfg["nginx_conf_path"], l.LoadNginxConfPath},
		// 队列配置
		{"queue_max_retry", cfg["queue_max_retry"], l.LoadQueueMaxRetry},
		{"queue_task_timeout", cfg["queue_task_timeout"], l.LoadQueueTaskTimeout},
		{"queue_retry_delay_base", cfg["queue_retry_delay_base"], l.LoadQueueRetryDelayBase},
		// 服务配置
		{"service_name", cfg["service_name"], l.LoadServiceName},
		{"service_description", cfg["service_description"], l.LoadServiceDescription},
	}

	for _, c := range configs {
		cfg[c.key] = c.loader(c.defaultValue)
	}
}
