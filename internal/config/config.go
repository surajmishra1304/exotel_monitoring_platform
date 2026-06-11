package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server    ServerConfig
	MySQL     MySQLConfig
	Redis     RedisConfig
	Scheduler SchedulerConfig
	Slack     SlackConfig
	Email     EmailConfig
	Crypto    CryptoConfig
	Alerts    AlertsConfig
	Exotel    ExotelConfig
}

type ServerConfig struct {
	Port                 int `mapstructure:"port"`
	ReadTimeoutSeconds   int `mapstructure:"read_timeout_seconds"`
	WriteTimeoutSeconds  int `mapstructure:"write_timeout_seconds"`
	IdleTimeoutSeconds   int `mapstructure:"idle_timeout_seconds"`
}

type MySQLConfig struct {
	Host                string `mapstructure:"host"`
	Port                int    `mapstructure:"port"`
	Username            string `mapstructure:"username"`
	Password            string `mapstructure:"password"`
	Database            string `mapstructure:"database"`
	MaxOpenConns        int    `mapstructure:"max_open_conns"`
	MaxIdleConns        int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetimeHours int   `mapstructure:"conn_max_lifetime_hours"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type SchedulerConfig struct {
	WorkerPoolSize      int  `mapstructure:"worker_pool_size"`
	PollIntervalSeconds int  `mapstructure:"poll_interval_seconds"`
	SkipCallLogsWrite   bool `mapstructure:"skip_call_logs_write"`
}

type SlackConfig struct {
	WebhookURL string `mapstructure:"webhook_url"`
	Enabled    bool   `mapstructure:"enabled"`
}

type EmailConfig struct {
	SMTPHost string `mapstructure:"smtp_host"`
	SMTPPort int    `mapstructure:"smtp_port"`
	From     string `mapstructure:"from"`
	Password string `mapstructure:"password"`
	Enabled  bool   `mapstructure:"enabled"`
}

type CryptoConfig struct {
	SecretKey string `mapstructure:"secret_key"`
}

type AlertsConfig struct {
	HighLatencyThresholdMs int `mapstructure:"high_latency_threshold_ms"`
	MaxRetries             int `mapstructure:"max_retries"`
	RetryBaseDelayMs       int `mapstructure:"retry_base_delay_ms"`
	RetryMaxDelayMs        int `mapstructure:"retry_max_delay_ms"`
}

type ExotelConfig struct {
	BaseURL        string `mapstructure:"base_url"`
	HeartbeatURL   string `mapstructure:"heartbeat_url"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
}

var App Config

// Load reads config.yaml and binds environment variable overrides.
func Load() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Allow ENV overrides like MYSQL_HOST, REDIS_PORT, etc.
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error reading config: %w", err))
	}

	if err := viper.Unmarshal(&App); err != nil {
		panic(fmt.Errorf("fatal error unmarshaling config: %w", err))
	}
}
