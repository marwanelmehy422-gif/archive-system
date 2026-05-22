package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort  string
	AppEnv   string
	Database DatabaseConfig
	JWT      JWTConfig
	Upload   UploadConfig
}

type DatabaseConfig struct {
	Host           string
	Port           string
	Name           string
	User           string
	Password       string
	SSLMode        string
	MaxConnections int
}

type JWTConfig struct {
	Secret      string
	ExpiryHours int
}

type UploadConfig struct {
	Dir           string
	MaxFileSizeMB int64
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	maxConn, _ := strconv.Atoi(getEnv("DB_MAX_CONNECTIONS", "20"))
	jwtExpiry, _ := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "24"))
	maxFileMB, _ := strconv.ParseInt(getEnv("MAX_FILE_SIZE_MB", "100"), 10, 64)

	cfg := &Config{
		AppPort: getEnv("APP_PORT", "8080"),
		AppEnv:  getEnv("APP_ENV", "development"),
		Database: DatabaseConfig{
			Host:           getEnv("DB_HOST", "localhost"),
			Port:           getEnv("DB_PORT", "5432"),
			Name:           getEnv("DB_NAME", "archive_db"),
			User:           getEnv("DB_USER", "postgres"),
			Password:       getEnv("DB_PASSWORD", ""),
			SSLMode:        getEnv("DB_SSLMODE", "disable"),
			MaxConnections: maxConn,
		},
		JWT: JWTConfig{
			Secret:      getEnv("JWT_SECRET", ""),
			ExpiryHours: jwtExpiry,
		},
		Upload: UploadConfig{
			Dir:           getEnv("UPLOAD_DIR", "./uploads"),
			MaxFileSizeMB: maxFileMB,
		},
	}

	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s pool_max_conns=%d",
		d.Host, d.Port, d.Name, d.User, d.Password, d.SSLMode, d.MaxConnections,
	)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
