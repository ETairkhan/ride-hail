package config

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Структура для хранения конфигурации
type Config struct {
	DBConfig        DBConfig
	RabbitMQConfig  RabbitMQConfig
	WebSocketConfig WebSocketConfig
	ServicesConfig  ServicesConfig
}

type DBConfig struct {
	Host      string
	Port      string
	User      string
	Password  string
	Name      string
	JWTSecret string
	JWTExpiry int
}

type RabbitMQConfig struct {
	Host     string
	Port     string
	User     string
	Password string
}

type WebSocketConfig struct {
	Port string
}

type ServicesConfig struct {
	RideServicePort           string
	DriverLocationServicePort string
	AdminServicePort          string
	AuthServicePort           string
}

// Загрузка конфигурации из переменных окружения
func LoadConfig() Config {
	defaultValues := GetDefaultValue()
	dbConfig := DBConfig{
		Host:      getEnv("DB_HOST", defaultValues["DB_HOST"]),
		Port:      getEnv("DB_PORT", defaultValues["DB_PORT"]),
		User:      getEnv("DB_USER", defaultValues["DB_USER"]),
		Password:  getEnv("DB_PASSWORD", defaultValues["DB_PASSWORD"]),
		Name:      getEnv("DB_NAME", defaultValues["DB_NAME"]),
		JWTSecret: getEnv("JWT_SECRET", defaultValues["JWT_SECRET"]),
		JWTExpiry: getEnvAsInt("JWT_EXPIRY_HOURS", defaultValues["JWT_EXPIRY_HOURS"]),
	}
	rabbitMQConfig := RabbitMQConfig{
		Host:     getEnv("RABBITMQ_HOST", defaultValues["RABBITMQ_HOST"]),
		Port:     getEnv("RABBITMQ_PORT", defaultValues["RABBITMQ_PORT"]),
		User:     getEnv("RABBITMQ_USER", defaultValues["RABBITMQ_USER"]),
		Password: getEnv("RABBITMQ_PASSWORD", defaultValues["RABBITMQ_PASSWORD"]),
	}
	webSocketConfig := WebSocketConfig{
		Port: getEnv("WS_PORT", defaultValues["WS_PORT"]),
	}
	servicesConfig := ServicesConfig{
		RideServicePort:           getEnv("RIDE_SERVICE_PORT", defaultValues["RIDE_SERVICE_PORT"]),
		DriverLocationServicePort: getEnv("DRIVER_LOCATION_SERVICE_PORT", defaultValues["DRIVER_LOCATION_SERVICE_PORT"]),
		AdminServicePort:          getEnv("ADMIN_SERVICE_PORT", defaultValues["ADMIN_SERVICE_PORT"]),
		AuthServicePort:           getEnv("AUTH_SERVICE_PORT", defaultValues["AUTH_SERVICE_PORT"]),
	}
	return Config{
		DBConfig:        dbConfig,
		RabbitMQConfig:  rabbitMQConfig,
		WebSocketConfig: webSocketConfig,
		ServicesConfig:  servicesConfig,
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

// Получение значения из переменной окружения как int с дефолтным значением
func getEnvAsInt(key, defaultValue string) int {
	value := os.Getenv(key)
	intDefaultValue, err := strconv.Atoi(defaultValue)
	if err != nil {
		return 24
	}
	if value == "" {
		return intDefaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return intDefaultValue
	}
	return intValue
}

// GetDefaultValue создает карту с ключами и их дефолтными значениями из config.yaml
func GetDefaultValue() map[string]string {
	file, err := os.Open("config.yaml")
	if err != nil {
		return make(map[string]string)
	}
	defer file.Close()

	defaultValues := make(map[string]string)
	scanner := bufio.NewScanner(file)
	re := regexp.MustCompile(`\$\{([^:]+):-([^}]+)\}`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := re.FindStringSubmatch(line)
		if len(matches) == 3 {
			defaultValues[matches[1]] = matches[2]
		}
	}

	return defaultValues
}
