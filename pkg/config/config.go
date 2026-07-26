package config

import (
	"fiction-book/pkg/payment"

	"github.com/spf13/viper"
)

// Config 配置结构
type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	Database      MysqlConfig         `mapstructure:"mysql"`
	Elasticsearch ElasticsearchConfig `mapstructure:"elasticsearch"`
	Redis         RedisConfig         `mapstructure:"redis"`
	JWT           JWTConfig           `mapstructure:"jwt"`
	Pprof         PprofConfig         `mapstructure:"pprof"`
	Sentry        SentryConfig        `mapstructure:"sentry"`
	Tracing       TracingConfig       `mapstructure:"tracing"`
	OSS           OSSConfig           `mapstructure:"oss"`
	Resend        Resend              `mapstructure:"resend"`
	Kafka         KafkaConfig         `mapstructure:"kafka"`
	Purchased     PurchasedConfig     `mapstructure:"purchased"`
	Payment       payment.Config      `mapstructure:"payment"`
}

// OSSConfig 对象存储配置
type OSSConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	BucketName      string `mapstructure:"bucket_name"`
	Region          string `mapstructure:"region"`
	RoleARN         string `mapstructure:"role_arn"`
	RoleSession     string `mapstructure:"role_session"`
	CDNURL          string `mapstructure:"cdn_url"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Website      string `mapstructure:"website"`
	Port         int    `mapstructure:"port"`
	Mode         string `mapstructure:"mode"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
}

// MysqlConfig 数据库配置
type MysqlConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Database        string `mapstructure:"database"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

// ElasticsearchConfig 数据库配置
type ElasticsearchConfig struct {
	Address  []string `mapstructure:"address"`
	Username string   `mapstructure:"username"`
	Password string   `mapstructure:"password"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Expire int    `mapstructure:"expire"`
}

// PprofConfig Pprof 性能分析配置
type PprofConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// SentryConfig Sentry 错误追踪配置
type SentryConfig struct {
	Enabled          bool    `mapstructure:"enabled"`
	DSN              string  `mapstructure:"dsn"`
	Environment      string  `mapstructure:"environment"`
	TracesSampleRate float64 `mapstructure:"traces_sample_rate"`
	Debug            bool    `mapstructure:"debug"`
}

// TracingConfig OpenTelemetry 配置
type TracingConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	ServiceName    string `mapstructure:"service_name"`
	JaegerEndpoint string `mapstructure:"jaeger_endpoint"`
}

type Resend struct {
	From   string `mapstructure:"from"`
	ApiKey string `mapstructure:"api_key"`
}

type KafkaConfig struct {
	Addr  []string         `mapstructure:"addr"`
	Topic TopicKafkaConfig `mapstructure:"topic"`
}

type TopicKafkaConfig struct {
	DatabaseSync string `mapstructure:"database_sync"`
}

type PurchasedConfig struct {
	VIP1 VIPPriceConfig `mapstructure:"vip_1"`
	VIP2 VIPPriceConfig `mapstructure:"vip_2"`
}

type VIPPriceConfig struct {
	Month int64 `mapstructure:"month"`
	Year  int64 `mapstructure:"year"`
}

// Load 加载配置
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	// 支持环境变量覆盖
	viper.AutomaticEnv()

	// 尝试读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// LoadWithFile 从指定文件加载配置
func LoadWithFile(configFile string) (*Config, error) {
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")

	// 支持环境变量覆盖
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
