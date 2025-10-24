package config

import (
	"os"
	"strconv"
)

type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type RabbitMQConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type WebSocketConfig struct {
	Port int `json:"port"`
}

type ServicesConfig struct {
	DriverLocationService int `json:"driver_location_service"`
}

type Config struct {
	Database  DatabaseConfig  `json:"database"`
	RabbitMQ  RabbitMQConfig  `json:"rabbitmq"`
	WebSocket WebSocketConfig `json:"websocket"`
	Services  ServicesConfig  `json:"services"`
}

func Load() (*Config, error) {
	// Set default values first
	cfg := &Config{
		Database: DatabaseConfig{
			Host:     "postgres",
			Port:     5432,
			User:     "ridehail_user",
			Password: "ridehail_pass",
			Database: "ridehail_db",
		},
		RabbitMQ: RabbitMQConfig{
			Host:     "rabbitmq",
			Port:     5672,
			User:     "guest",
			Password: "guest",
		},
		WebSocket: WebSocketConfig{
			Port: 8080,
		},
		Services: ServicesConfig{
			DriverLocationService: 3001,
		},
	}

	// Database - override with environment variables
	if host := os.Getenv("DB_HOST"); host != "" {
		cfg.Database.Host = host
	}
	if port := os.Getenv("DB_PORT"); port != "" {
		portInt, err := strconv.Atoi(port)
		if err == nil {
			cfg.Database.Port = portInt
		}
	}
	if user := os.Getenv("DB_USER"); user != "" {
		cfg.Database.User = user
	}
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		cfg.Database.Password = password
	}
	if database := os.Getenv("DB_NAME"); database != "" {
		cfg.Database.Database = database
	}

	// RabbitMQ - override with environment variables
	if rabbitHost := os.Getenv("RABBITMQ_HOST"); rabbitHost != "" {
		cfg.RabbitMQ.Host = rabbitHost
	}
	if rabbitPort := os.Getenv("RABBITMQ_PORT"); rabbitPort != "" {
		rabbitPortInt, err := strconv.Atoi(rabbitPort)
		if err == nil {
			cfg.RabbitMQ.Port = rabbitPortInt
		}
	}
	if rabbitUser := os.Getenv("RABBITMQ_USER"); rabbitUser != "" {
		cfg.RabbitMQ.User = rabbitUser
	}
	if rabbitPassword := os.Getenv("RABBITMQ_PASSWORD"); rabbitPassword != "" {
		cfg.RabbitMQ.Password = rabbitPassword
	}

	// WebSocket
	if wsPort := os.Getenv("WS_PORT"); wsPort != "" {
		wsPortInt, err := strconv.Atoi(wsPort)
		if err == nil {
			cfg.WebSocket.Port = wsPortInt
		}
	}

	// Services
	if driverLocService := os.Getenv("DRIVER_LOCATION_SERVICE_PORT"); driverLocService != "" {
		driverLocServiceInt, err := strconv.Atoi(driverLocService)
		if err == nil {
			cfg.Services.DriverLocationService = driverLocServiceInt
		}
	}

	return cfg, nil
}
