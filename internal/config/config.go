package config

import (
	"github.com/spf13/viper"
	"strings"
	"time"
)

// Config is the main struct that holds all configuration for the application.
type Config struct {
	Logger    LoggerConfig    `mapstructure:"logger"`
	HTTP      HTTPConfig      `mapstructure:"http"`
	Postgres  PostgresConfig  `mapstructure:"postgres"`
	RabbitMQ  RabbitMQConfig  `mapstructure:"rabbitmq"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Notifiers NotifiersConfig `mapstructure:"notifiers"`
}

// LoggerConfig holds logging-specific settings.
type LoggerConfig struct {
	Level string `mapstructure:"level"`
}

// HTTPConfig holds HTTP server-specific settings.
type HTTPConfig struct {
	Port    string `mapstructure:"port"`
	GinMode string `mapstructure:"gin_mode"`
}

// PostgresConfig holds all settings for the PostgreSQL database connection.
type PostgresConfig struct {
	MasterDSN string     `mapstructure:"master_dsn"`
	SlaveDSNs []string   `mapstructure:"slave_dsns"`
	Pool      PoolConfig `mapstructure:"pool"`
}

// PoolConfig defines the connection pool settings for the database.
type PoolConfig struct {
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// RabbitMQConfig holds all settings for the RabbitMQ connection.
type RabbitMQConfig struct {
	DSN string `mapstructure:"dsn"`
}

// RedisConfig holds all settings for the Redis connection.
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// NotifiersConfig holds configurations for all notification channels.
type NotifiersConfig struct {
	// Mode can be "development" or "production".
	// In "development" mode, all notifiers will be replaced by the LogNotifier.
	Mode     string         `mapstructure:"mode"`
	Email    EmailConfig    `mapstructure:"email"`
	Telegram TelegramConfig `mapstructure:"telegram"`
}

// EmailConfig holds SMTP settings for the email notifier.
type EmailConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
}

// TelegramConfig holds settings for the Telegram notifier.
type TelegramConfig struct {
	BotToken string `mapstructure:"bot_token"`
}

// NewConfig parses the YAML file and environment variables to return a configuration struct.
func NewConfig() (*Config, error) {
	v := viper.New()

	v.SetConfigFile("configs/config.yaml")

	v.SetDefault("logger.level", "info")
	v.SetDefault("http.port", ":8080")
	v.SetDefault("http.gin_mode", "release")
	v.SetDefault("notifiers.mode", "log_only")

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
