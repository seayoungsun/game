package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	Server           ServerConfig           `mapstructure:"server"`
	Database         DatabaseConfig         `mapstructure:"database"`
	Redis            RedisConfig            `mapstructure:"redis"`
	ES               ESConfig               `mapstructure:"elasticsearch"`
	JWT              JWTConfig              `mapstructure:"jwt"`
	Log              LogConfig              `mapstructure:"log"`
	Payment          PaymentConfig          `mapstructure:"payment"`
	ServiceDiscovery ServiceDiscoveryConfig `mapstructure:"service_discovery"`
	Kafka            KafkaConfig            `mapstructure:"kafka"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Mode         string `mapstructure:"mode"`         // debug, release, test
	Port         int    `mapstructure:"port"`         // API服务端口
	GamePort     int    `mapstructure:"game_port"`    // 游戏服务器端口
	AdminPort    int    `mapstructure:"admin_port"`   // 管理后台端口（默认8082）
	MachineID    int    `mapstructure:"machine_id"`   // 机器ID（0-1023，用于雪花算法）
	ReadTimeout  int    `mapstructure:"read_timeout"` // 秒
	WriteTimeout int    `mapstructure:"write_timeout"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxLifetime  int    `mapstructure:"max_lifetime"` // 秒
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

// ESConfig Elasticsearch配置
type ESConfig struct {
	Addresses []string `mapstructure:"addresses"`
	Username  string   `mapstructure:"username"`
	Password  string   `mapstructure:"password"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	Expiration int    `mapstructure:"expiration"` // 小时
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`       // debug, info, warn, error
	OutputPath string `mapstructure:"output_path"` // 日志输出路径
	MaxSize    int    `mapstructure:"max_size"`    // MB
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"` // 天
}

// PaymentConfig 支付配置
type PaymentConfig struct {
	EtherscanAPIKey string `mapstructure:"etherscan_api_key"` // Etherscan API Key（用于ERC20查询）
	MasterMnemonic  string `mapstructure:"master_mnemonic"`   // 主钱包助记词（必须配置，用于HD钱包派生地址）
}

// ServiceDiscoveryConfig 服务发现配置
type ServiceDiscoveryConfig struct {
	Enabled             bool   `mapstructure:"enabled"`               // 是否启用服务发现
	Type                string `mapstructure:"type"`                  // 类型: "consul", "redis"
	ConsulAddr          string `mapstructure:"consul_addr"`           // Consul 地址 (如: localhost:8500)
	HealthCheckAddress  string `mapstructure:"health_check_address"`  // 健康检查地址（可选，为空则自动检测）
	HealthCheckInterval int    `mapstructure:"health_check_interval"` // 健康检查间隔（秒）
	HealthCheckTimeout  int    `mapstructure:"health_check_timeout"`  // 健康检查超时（秒）
	DeregisterAfter     int    `mapstructure:"deregister_after"`      // 故障后自动注销时间（秒）
	// Redis 相关配置（如果使用 Redis）
	InstanceTTL       int `mapstructure:"instance_ttl"`       // 实例 TTL（秒）
	HeartbeatInterval int `mapstructure:"heartbeat_interval"` // 心跳间隔（秒）
}

// KafkaConfig Kafka 配置
type KafkaConfig struct {
	Enabled       bool     `mapstructure:"enabled"`        // 是否启用 Kafka
	Brokers       []string `mapstructure:"brokers"`        // Kafka Broker 地址列表
	TopicPrefix   string   `mapstructure:"topic_prefix"`   // Topic 前缀
	ConsumerGroup string   `mapstructure:"consumer_group"` // 消费者组名称
	// Producer 配置
	ProducerAcks    string `mapstructure:"producer_acks"`    // acks: "0", "1", "all"
	ProducerRetries int    `mapstructure:"producer_retries"` // 重试次数
	BatchSize       int    `mapstructure:"batch_size"`       // 批量大小（字节）
	LingerMs        int    `mapstructure:"linger_ms"`        // 等待时间（毫秒）
	CompressionType string `mapstructure:"compression_type"` // 压缩类型: "none", "gzip", "snappy", "lz4"
	// Consumer 配置
	ConsumerAutoCommit     bool `mapstructure:"consumer_auto_commit"`      // 是否自动提交
	ConsumerMaxPollRecords int  `mapstructure:"consumer_max_poll_records"` // 每次拉取最大记录数
	FetchMinBytes          int  `mapstructure:"fetch_min_bytes"`           // 最小拉取字节数
	FetchMaxWaitMs         int  `mapstructure:"fetch_max_wait_ms"`         // 最大等待时间（毫秒）
}

var globalConfig *Config

type loadOptions struct {
	ConfigFile string
	Env        string
}

// Load 加载配置（兼容旧签名），自动根据 APP_ENV 选择环境配置
func Load(configPath string) (*Config, error) {
	return load(loadOptions{
		ConfigFile: configPath,
		Env:        os.Getenv("APP_ENV"),
	})
}

// LoadWithEnv 允许显式指定环境（local/dev/prod 等）
func LoadWithEnv(env string) (*Config, error) {
	return load(loadOptions{Env: env})
}

func load(opts loadOptions) (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	setDefaults(v)

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if opts.ConfigFile != "" {
		v.SetConfigFile(opts.ConfigFile)
		if err := v.ReadInConfig(); err != nil {
			log.Printf("Warning: 无法读取配置文件 %s: %v", opts.ConfigFile, err)
		}
	} else {
		v.AddConfigPath("./configs")
		v.AddConfigPath("../configs")
		v.AddConfigPath("../../configs")

		// 读取基础配置 config.yaml
		v.SetConfigName("config")
		if err := v.ReadInConfig(); err != nil {
			log.Printf("Warning: 未找到 config.yaml，使用默认配置: %v", err)
		}

		env := strings.TrimSpace(opts.Env)
		if env == "" {
			env = "local"
		}
		env = strings.ToLower(env)

		// 合并环境配置（config.<env>.yaml），不存在则忽略
		v.SetConfigName(fmt.Sprintf("config.%s", env))
		if err := v.MergeInConfig(); err != nil {
			log.Printf("Info: 未找到环境配置 config.%s.yaml，继续使用基础配置", env)
		}
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("配置解析失败: %w", err)
	}

	globalConfig = &config
	return &config, nil
}

// Get 获取全局配置
func Get() *Config {
	if globalConfig == nil {
		panic("配置未初始化，请先调用 config.Load()")
	}
	return globalConfig
}

// setDefaults 设置默认值
func setDefaults(v *viper.Viper) {
	// 服务器默认配置
	v.SetDefault("server.mode", "debug")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.game_port", 8081)
	v.SetDefault("server.read_timeout", 30)
	v.SetDefault("server.write_timeout", 30)

	// 数据库默认配置
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 3306)
	v.SetDefault("database.user", "root")
	v.SetDefault("database.password", "")
	v.SetDefault("database.database", "game_platform")
	v.SetDefault("database.max_open_conns", 100)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.max_lifetime", 3600)

	// Redis默认配置
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)

	// ES默认配置
	v.SetDefault("elasticsearch.addresses", []string{"http://localhost:9200"})

	// JWT默认配置
	v.SetDefault("jwt.secret", "your-secret-key-change-in-production")
	v.SetDefault("jwt.expiration", 24)

	// 日志默认配置
	v.SetDefault("log.level", "info")
	v.SetDefault("log.output_path", "./logs")
	v.SetDefault("log.max_size", 100)
	v.SetDefault("log.max_backups", 7)
	v.SetDefault("log.max_age", 30)

	// 服务发现默认配置
	v.SetDefault("service_discovery.enabled", false)
	v.SetDefault("service_discovery.type", "consul")
	v.SetDefault("service_discovery.consul_addr", "localhost:8500")
	v.SetDefault("service_discovery.health_check_interval", 10)
	v.SetDefault("service_discovery.health_check_timeout", 3)
	v.SetDefault("service_discovery.deregister_after", 30)
	v.SetDefault("service_discovery.instance_ttl", 60)
	v.SetDefault("service_discovery.heartbeat_interval", 30)

	// Kafka 默认配置
	v.SetDefault("kafka.enabled", false)
	v.SetDefault("kafka.brokers", []string{"localhost:9092"})
	v.SetDefault("kafka.topic_prefix", "game")
	v.SetDefault("kafka.consumer_group", "game-server-group")
	v.SetDefault("kafka.producer_acks", "all")
	v.SetDefault("kafka.producer_retries", 3)
	v.SetDefault("kafka.batch_size", 16384)
	v.SetDefault("kafka.linger_ms", 10)
	v.SetDefault("kafka.compression_type", "gzip")
	v.SetDefault("kafka.consumer_auto_commit", false)
	v.SetDefault("kafka.consumer_max_poll_records", 100)
	v.SetDefault("kafka.fetch_min_bytes", 1024)
	v.SetDefault("kafka.fetch_max_wait_ms", 100)
}
