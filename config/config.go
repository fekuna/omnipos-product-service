package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server   ServerConfig
	Logger   LoggerConfig
	Postgres PostgresConfig
	JWT      JWTConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
	Elastic  ElasticsearchConfig
}

type ServerConfig struct {
	AppEnv   string
	GRPCPort string
}

type LoggerConfig struct {
	Level             string
	Encoding          string
	DisableCaller     bool
	DisableStacktrace bool
}

type PostgresConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int
	ConnMaxIdleTime int
}

type JWTConfig struct {
	SecretKey string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type KafkaConfig struct {
	Brokers []string
	Topic   string
	GroupID string
}

type ElasticsearchConfig struct {
	Addresses []string
	Username  string
	Password  string
}

func LoadEnv() *Config {
	// Basic config loading
	// In a real scenario, use structured config loader like viper or koanf
	return &Config{
		Server: ServerConfig{
			AppEnv:   getEnv("APP_ENV", "dev"),
			GRPCPort: getEnv("GRPC_PORT", ":8082"),
		},
		Logger: LoggerConfig{
			Level:             getEnv("LOGGER_LEVEL", "debug"),
			Encoding:          getEnv("LOGGER_ENCODING", "console"),
			DisableCaller:     getEnvBool("LOGGER_DISABLE_CALLER", false),
			DisableStacktrace: getEnvBool("LOGGER_DISABLE_STACKTRACE", true),
		},
		Postgres: PostgresConfig{
			Host:            getEnv("POSTGRES_HOST", "localhost"),
			Port:            getEnv("POSTGRES_PORT", "5433"),
			User:            getEnv("POSTGRES_USER", "omnipos"),
			Password:        getEnv("POSTGRES_PASSWORD", "omnipos"),
			DBName:          getEnv("POSTGRES_DB", "omnipos_product"),
			SSLMode:         getEnv("POSTGRES_SSLMODE", "disable"),
			MaxOpenConns:    getEnvInt("POSTGRES_MAX_OPEN_CONNS", 10),
			MaxIdleConns:    getEnvInt("POSTGRES_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvInt("POSTGRES_CONN_MAX_LIFETIME", 300),
			ConnMaxIdleTime: getEnvInt("POSTGRES_CONN_MAX_IDLE_TIME", 60),
		},
		JWT: JWTConfig{
			SecretKey: getEnv("JWT_SECRET_KEY", "your-secret-key-change-this-in-prod"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Kafka: KafkaConfig{
			Brokers: getEnvSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			Topic:   getEnv("KAFKA_TOPIC_ORDERS", "orders.events"),
			GroupID: getEnv("KAFKA_GROUP_INVENTORY", "inventory"),
		},
		Elastic: ElasticsearchConfig{
			Addresses: getEnvSlice("ELASTICSEARCH_ADDRESSES", []string{"http://localhost:9200"}),
			Username:  getEnv("ELASTICSEARCH_USERNAME", ""),
			Password:  getEnv("ELASTICSEARCH_PASSWORD", ""),
		},
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return fallback
}

func getEnvSlice(key string, fallback []string) []string {
	if value, ok := os.LookupEnv(key); ok {
		return strings.Split(value, ",")
	}
	return fallback
}
