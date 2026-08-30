package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/joho/godotenv"
)

type Config struct {
	App AppConfig `mapstructure:"app"`
	Server ServerConfig `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Auth AuthConfig `mapstructure:"auth"`
	Logger LoggerConfig `mapstructure:"logger"`
}

type AppConfig struct {
	Name string `mapstructure:"name" validate:"required"`
	Environment string `mapstructure:"environment" validate:"required,oneof=development staging production test"`
}

type ServerConfig struct {
	Host string `mapstructure:"host" validate:"required"`
	Port int `mapstructure:"port" validate:"required,min=1,max=65535"`
	ReadTimeout time.Duration `mapstructure:"read_timeout" validate:"gt=0"`
	WriteTimeout time.Duration `mapstructure:"write_timeout" validate:"gt=0"`
	IdleTimeout time.Duration `mapstructure:"idle_timeout" validate:"gt=0"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" validate:"gt=0"`
}

type DatabaseConfig struct {
	Host string `mapstructure:"host" validate:"required"`
	Port int `mapstructure:"port" validate:"required,min=1,max=65535"`
	User string `mapstructure:"user" validate:"required"`
	Password string `mapstructure:"password" validate:"required"`
	Name string `mapstructure:"name" validate:"required"`
	SSLMode string `mapstructure:"ssl_mode" validate:"required,oneof=disable require verify-ca verify-full"`
	MaxConnections int32 `mapstructure:"max_connections" validate:"required,min=1"`
	MinConnections int32 `mapstructure:"min_connections" validate:"gte=0"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime" validate:"gt=0"`
	MaxConnIdleTime time.Duration `mapstructure:"max_conn_idle_time" validate:"gt=0"`
	HealthCheckPeriod time.Duration `mapstructure:"health_check_period" validate:"gt=0"`
}

type AuthConfig struct {
	AccessTokenSecret string `mapstructure:"access_token_secret" validate:"required,min=32"`
	RefreshTokenSecret string `mapstructure:"refresh_token_secret" validate:"required,min=32"`
	AccessTokenTTL time.Duration `mapstructure:"access_token_ttl" validate:"gt=0"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl" validate:"gt=0"`
	Issuer string `mapstructure:"issuer" validate:"required"`
}

type LoggerConfig struct {
	Level string `mapstructure:"level" validate:"required,oneof=debug info warn error"`
	Format string `mapstructure:"format" validate:"required,oneof=json text"`
	AddSource bool `mapstructure:"add_source"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/gorest")

	v.SetEnvPrefix("GOREST")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		var configFileNotFound viper.ConfigFileNotFoundError

		if ok := isConfigFileNotFound(err, &configFileNotFound); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name","gorest")
	v.SetDefault("app.environment", "development")

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 10*time.Second)
	v.SetDefault("server.write_timeout", 10*time.Second)
	v.SetDefault("server.idle_timeout", 60*time.Second)
	v.SetDefault("server.shutdown_timeout", 10*time.Second)
	
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_connections", 20)
	v.SetDefault("database.min_connections", 3)
	v.SetDefault("database.max_conn_lifetime", time.Hour)
	v.SetDefault("database.max_conn_idle_time", 30*time.Minute)
	v.SetDefault("database.health_check_period", time.Minute)

	v.SetDefault("auth.access_token_ttl", 15*time.Minute)
	v.SetDefault("auth.refresh_token_ttl", 7*24*time.Hour)
	v.SetDefault("auth.issuer", "gorest")

	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "json")
	v.SetDefault("logger.add_source", false)
}

func isConfigFileNotFound(err error, target *viper.ConfigFileNotFoundError) bool {
	return errors.As(err, target)
}
