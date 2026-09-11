package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig      `envPrefix:"GOREST_APP_"`
	Server   ServerConfig   `envPrefix:"GOREST_SERVER_"`
	Database DatabaseConfig `envPrefix:"GOREST_DATABASE_"`
	Auth     AuthConfig     `envPrefix:"GOREST_AUTH_"`
	CORS     CORSConfig     `envPrefix:"GOREST_CORS_"`
	Logger   LoggerConfig   `envPrefix:"GOREST_LOGGER_"`
}

type AppConfig struct {
	Name        string `env:"NAME,required"`
	Environment string `env:"ENVIRONMENT,required"`
}

type ServerConfig struct {
	Host            string        `env:"HOST,required"`
	Port            int           `env:"PORT,required"`
	ReadTimeout     time.Duration `env:"READ_TIMEOUT,required"`
	WriteTimeout    time.Duration `env:"WRITE_TIMEOUT,required"`
	IdleTimeout     time.Duration `env:"IDLE_TIMEOUT,required"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT,required"`
}

type DatabaseConfig struct {
	Host              string        `env:"HOST,required"`
	Port              int           `env:"PORT,required"`
	User              string        `env:"USER,required"`
	Password          string        `env:"PASSWORD,required"`
	Name              string        `env:"NAME,required"`
	SSLMode           string        `env:"SSL_MODE,required"`
	MaxConnections    int32         `env:"MAX_CONNECTIONS,required"`
	MinConnections    int32         `env:"MIN_CONNECTIONS,required"`
	MaxConnLifetime   time.Duration `env:"MAX_CONN_LIFETIME,required"`
	MaxConnIdleTime   time.Duration `env:"MAX_CONN_IDLE_TIME,required"`
	HealthCheckPeriod time.Duration `env:"HEALTH_CHECK_PERIOD,required"`
}

type AuthConfig struct {
	AccessTokenSecret  string        `env:"ACCESS_TOKEN_SECRET,required"`
	RefreshTokenSecret string        `env:"REFRESH_TOKEN_SECRET,required"`
	AccessTokenTTL     time.Duration `env:"ACCESS_TOKEN_TTL,required"`
	RefreshTokenTTL    time.Duration `env:"REFRESH_TOKEN_TTL,required"`
	Issuer             string        `env:"ISSUER,required"`
	BcryptCost         int           `env:"BCRYPT_COST,required"`
	CookieSecure       bool          `env:"COOKIE_SECURE,required"`
}

type CORSConfig struct {
	AllowedOrigins []string `envPrefix:"ALLOWED_ORIGINS" envSeparator:","`
}

type LoggerConfig struct {
	Level     string `env:"LEVEL,required"`
	Format    string `env:"FORMAT,required"`
	AddSource bool   `env:"ADD_SOURCE,required"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse environment: %w", err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

func validateConfig(cfg *Config) error {
	v := validator.New()

	if err := v.Struct(cfg); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	return nil
}
