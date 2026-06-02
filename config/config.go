package config

import (
	"strings"
	"time"

	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/spf13/viper"
)

type Config struct {
	DatabaseURL  string
	Port         string
	GitLabToken  string
	GitLabURL    string
	HarborURL    string
	HarborUser   string
	HarborPass   string
	WorkerPeriod time.Duration
	SecretsPath  string
	Logging      types.LoggingConfig

	JenkinsURL  string
	JenkinsUser string
	JenkinsPass string
	
	// Nacos 配置中心（可选）
	NacosServerAddr string // Nacos 服务器地址，如 "192.168.192.87:8848"
	NacosNamespace  string // 命名空间 ID，如 "zebra-dev"
	NacosUsername   string // Nacos 用户名
	NacosPassword   string // Nacos 密码
	NacosGroup      string // 配置分组，默认 DEFAULT_GROUP

	// Redis & Asynq 任务队列
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
	WorkerConcurrency int
}

func Load() *Config {
	// 设置配置文件名称和路径
	viper.SetConfigName("configs")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")

	// 设置环境变量前缀
	viper.SetEnvPrefix("ZEBRA")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// 兼容显式前缀变量与历史脚本中的裸 NACOS_* 变量。
	viper.BindEnv("nacos.server_addr", "ZEBRA_NACOS_SERVER_ADDR", "NACOS_SERVER_ADDR")
	viper.BindEnv("nacos.namespace", "ZEBRA_NACOS_NAMESPACE", "NACOS_NAMESPACE")
	viper.BindEnv("nacos.username", "ZEBRA_NACOS_USERNAME", "NACOS_USERNAME")
	viper.BindEnv("nacos.password", "ZEBRA_NACOS_PASSWORD", "NACOS_PASSWORD")
	viper.BindEnv("nacos.group", "ZEBRA_NACOS_GROUP", "NACOS_GROUP")

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		// 如果配置文件不存在，继续使用环境变量和默认值
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			panic(err)
		}
	}

	// 默认值设置
	viper.SetDefault("app.Port", "4123")
	viper.SetDefault("app.GitLabURL", "https://gitlab.com")
	viper.SetDefault("app.HarborURL", "registry.cn-shanghai.aliyuncs.com")

	// 注意：YAML配置中使用的是"5m"这样的字符串格式，需要特殊处理
	workerPeriodStr := viper.GetString("app.WorkerPeriod")
	var workerPeriod time.Duration
	if workerPeriodStr != "" {
		// 解析持续时间字符串，如"5m"
		workerPeriod, _ = time.ParseDuration(workerPeriodStr)
	} else {
		// 默认值10秒
		workerPeriod = 10 * time.Second
	}

	// Redis & worker 默认值
	viper.SetDefault("redis.addr", "127.0.0.1:6379")
	viper.SetDefault("worker.concurrency", 3)
	viper.BindEnv("redis.addr", "ZEBRA_REDIS_ADDR")
	viper.BindEnv("redis.password", "ZEBRA_REDIS_PASSWORD")

	// 设置日志默认值
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.encoding", "json")
	viper.SetDefault("logging.output_paths", []string{"stdout"})
	viper.SetDefault("logging.error_output_paths", []string{"stderr"})
	
	// Nacos 默认值
	viper.SetDefault("nacos.username", "nacos")
	viper.SetDefault("nacos.password", "nacos")
	viper.SetDefault("nacos.group", "DEFAULT_GROUP")

	cfg := &Config{
		DatabaseURL:  viper.GetString("app.DatabaseURL"),
		Port:         viper.GetString("app.Port"),
		GitLabToken:  viper.GetString("app.GitLabToken"),
		GitLabURL:    viper.GetString("app.GitLabURL"),
		HarborURL:    viper.GetString("app.HarborURL"),
		HarborUser:   viper.GetString("harbor.username"),
		HarborPass:   viper.GetString("harbor.password"),
		JenkinsURL:   viper.GetString("app.JenkinsURL"),
		JenkinsUser:  viper.GetString("app.JenkinsUser"),
		JenkinsPass:  viper.GetString("app.JenkinsPass"),
		WorkerPeriod: workerPeriod,
		SecretsPath:  viper.GetString("app.SecretsPath"),
		
		// Nacos 配置
		NacosServerAddr: viper.GetString("nacos.server_addr"),
		NacosNamespace:  viper.GetString("nacos.namespace"),
		NacosUsername:   viper.GetString("nacos.username"),
		NacosPassword:   viper.GetString("nacos.password"),
		NacosGroup:      viper.GetString("nacos.group"),

		// Redis & Asynq
		RedisAddr:         viper.GetString("redis.addr"),
		RedisPassword:     viper.GetString("redis.password"),
		RedisDB:           viper.GetInt("redis.db"),
		WorkerConcurrency: viper.GetInt("worker.concurrency"),

		Logging: types.LoggingConfig{
			Level:            viper.GetString("logging.level"),
			Encoding:         viper.GetString("logging.encoding"),
			OutputPaths:      viper.GetStringSlice("logging.output_paths"),
			ErrorOutputPaths: viper.GetStringSlice("logging.error_output_paths"),
			MaxSize:          viper.GetInt("logging.max_size"),
			MaxAge:           viper.GetInt("logging.max_age"),
			MaxBackups:       viper.GetInt("logging.max_backups"),
			Compress:         viper.GetBool("logging.compress"),
		},
	}

	return cfg
}
