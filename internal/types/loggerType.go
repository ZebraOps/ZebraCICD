package types

type LoggingConfig struct {
	Level            string   `mapstructure:"level"`
	Encoding         string   `mapstructure:"encoding"`
	OutputPaths      []string `mapstructure:"output_paths"`
	ErrorOutputPaths []string `mapstructure:"error_output_paths"`
	MaxSize          int      `mapstructure:"max_size"`
	MaxAge           int      `mapstructure:"max_age"`
	MaxBackups       int      `mapstructure:"max_backups"`
	Compress         bool     `mapstructure:"compress"`
}
