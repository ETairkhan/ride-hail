package rabbitmq

import (
	"fmt"
	"log"
	"ride-hail/internal/common/config"

	"github.com/rabbitmq/amqp091-go"
)

// Инициализация соединения с RabbitMQ
func InitRabbitMQ(config config.RabbitMQConfig) (*amqp091.Connection, error) {
	// Формирование строки подключения с использованием данных из конфигурации
	connStr := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		config.User, config.Password, config.Host, config.Port)

	// Устанавливаем соединение с RabbitMQ
	conn, err := amqp091.Dial(connStr)
	if err != nil {
		// Если соединение не удалось, возвращаем ошибку
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	// Логируем успешное подключение
	log.Println("Connected to RabbitMQ")

	return conn, nil
}
