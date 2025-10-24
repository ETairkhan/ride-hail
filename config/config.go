package config

import (
	"os"
	"strconv"
)

// Структура для хранения конфигурации
type Config struct {
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
	RabbitMQHost     string
	RabbitMQPort     string
	RabbitMQUser     string
	RabbitMQPassword string
	RideServicePort  string
	// JWT Configuration
	JWTSecret string `yaml:"jwt_secret"`
	JWTExpiry int    `yaml:"jwt_expiry_hours"`
}

// Загрузка конфигурации из переменных окружения
func LoadConfig() Config {
	return Config{
		DBHost:           getEnv("DB_HOST", "localhost"),
		DBPort:           getEnv("DB_PORT", "5432"),
		DBUser:           getEnv("DB_USER", "ridehail_user"),
		DBPassword:       getEnv("DB_PASSWORD", "ridehail_pass"),
		DBName:           getEnv("DB_NAME", "ridehail_db"),
		RabbitMQHost:     getEnv("RABBITMQ_HOST", "localhost"),
		RabbitMQPort:     getEnv("RABBITMQ_PORT", "5672"),
		RabbitMQUser:     getEnv("RABBITMQ_USER", "guest"),
		RabbitMQPassword: getEnv("RABBITMQ_PASSWORD", "guest"),
		RideServicePort:  getEnv("RIDE_SERVICE_PORT", "3000"),
		// JWT Configuration
		JWTSecret: getEnv("JWT_SECRET", "your-super-secret-jwt-key-minimum-32-chars-here"),
		JWTExpiry: getEnvAsInt("JWT_EXPIRY_HOURS", 24),
	}
}

// Получение значения из переменной окружения с дефолтным значением
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
