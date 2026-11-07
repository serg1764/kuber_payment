package config

import "os"

type Config struct {
	AppPort              string
	AppVersion           string
	NotificationGRPCAddr string
	DBHost               string
	DBPort               string
	DBName               string
	DBUser               string
	DBPassword           string
}

func Load() Config {
	return Config{
		AppPort:              getEnv("APP_PORT", "8080"),
		AppVersion:           getEnv("APP_VERSION", "v1-config.go"),
		NotificationGRPCAddr: getEnv("NOTIFICATION_GRPC_ADDR", "localhost:9090"),
		DBHost:               getEnv("DB_HOST", "localhost"),
		DBPort:               getEnv("DB_PORT", "5432"),
		DBName:               getEnv("DB_NAME", "payments_db"),
		DBUser:               getEnv("DB_USER", "payment_user"),
		DBPassword:           getEnv("DB_PASSWORD", "payment_password"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
