package utils

import (
	"os"
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
