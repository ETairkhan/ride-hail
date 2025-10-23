package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

type DatabaseConfig struct {
	Host     string `yaml:"host" env:"DB_HOST" env-default:"localhost"`
	Port     int    `yaml:"port" env:"DB_PORT" env-default:"5432"`
	User     string `yaml:"user" env:"DB_USER" env-default:"ridehail_user"`
	Password string `yaml:"password" env:"DB_PASSWORD" env-default:"ridehail_pass"`
	Database string `yaml:"database" env:"DB_NAME" env-default:"ridehail_db"`
}

type RabbitMQConfig struct {
	Host     string `yaml:"host" env:"RABBITMQ_HOST" env-default:"localhost"`
	Port     int    `yaml:"port" env:"RABBITMQ_PORT" env-default:"5672"`
	User     string `yaml:"user" env:"RABBITMQ_USER" env-default:"guest"`
	Password string `yaml:"password" env:"RABBITMQ_PASSWORD" env-default:"guest"`
}

type WebSocketConfig struct {
	Port int `yaml:"port" env:"WS_PORT" env-default:"8080"`
}

type ServicesConfig struct {
	DriverLocationService int `yaml:"driver_location_service" env:"DRIVER_LOCATION_SERVICE_PORT" env-default:"3001"`
}

type Config struct {
	Database  DatabaseConfig  `yaml:"database"`
	RabbitMQ  RabbitMQConfig  `yaml:"rabbitmq"`
	WebSocket WebSocketConfig `yaml:"websocket"`
	Services  ServicesConfig  `yaml:"services"`
}

func Load() (*Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Override with environment variables if present
	if host := os.Getenv("DB_HOST"); host != "" {
		cfg.Database.Host = host
	}
	if port := os.Getenv("DB_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Database.Port)
	}

	return &cfg, nil
}
