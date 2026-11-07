package config

import "os"

// Config хранит конфигурацию сервиса и подключения к БД.
type Config struct {
	GRPCPort   string
	AppVersion string
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string
}

// Load загружает конфигурацию из переменных окружения с дефолтными значениями.
func Load() Config {
	return Config{
		GRPCPort:   getEnv("GRPC_PORT", "9090"),
		AppVersion: getEnv("APP_VERSION", "v1"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     getEnv("DB_NAME", "notifications_db"),
		DBUser:     getEnv("DB_USER", "notification_user"),
		DBPassword: getEnv("DB_PASSWORD", "notification_password"),
	}
}

// getEnv возвращает значение переменной окружения или дефолт, если переменная не задана.
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	// Если переменная окружения пустая, используем значение по умолчанию.
	if value == "" {
		return defaultValue
	}
	return value
}
