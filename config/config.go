package config

import (
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
	WorkerPeriod time.Duration
	SecretsPath  string
	Logging      types.LoggingConfig

	JenkinsURL  string
	JenkinsUser string
	JenkinsPass string
}

func Load() *Config {
	// 设置配置文件名称和路径
	viper.SetConfigName("configs")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")

	// 设置环境变量前缀
	viper.SetEnvPrefix("ZEBRA")
	viper.AutomaticEnv()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		// 如果配置文件不存在，继续使用环境变量和默认值
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			panic(err)
		}
	}

	// 默认值设置
	viper.SetDefault("app.Port", "9527")
	viper.SetDefault("app.GitLabURL", "https://gitlab.com")
	viper.SetDefault("app.HarborURL", "")

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

	// 设置日志默认值
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.encoding", "json")
	viper.SetDefault("logging.output_paths", []string{"stdout"})
	viper.SetDefault("logging.error_output_paths", []string{"stderr"})

	cfg := &Config{
		DatabaseURL:  viper.GetString("app.DatabaseURL"),
		Port:         viper.GetString("app.Port"),
		GitLabToken:  viper.GetString("app.GitLabToken"),
		GitLabURL:    viper.GetString("app.GitLabURL"),
		HarborURL:    viper.GetString("app.HarborURL"),
		JenkinsURL:   viper.GetString("app.JenkinsURL"),
		JenkinsUser:  viper.GetString("app.JenkinsUser"),
		JenkinsPass:  viper.GetString("app.JenkinsPass"),
		WorkerPeriod: workerPeriod,
		SecretsPath:  viper.GetString("app.SecretsPath"),
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
