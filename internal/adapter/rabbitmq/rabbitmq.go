package rabbitmq

import (
	"fmt"
	"log"
	"ride-hail/internal/common/config"

	"github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	Conn *amqp091.Connection
}

// Initialize RabbitMQ connection and setup exchanges
func InitRabbitMQ(config config.RabbitMQConfig) (*amqp091.Connection, error) {
	connStr := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		config.User, config.Password, config.Host, config.Port)

	conn, err := amqp091.Dial(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	log.Println("Connected to RabbitMQ")

	// Setup exchanges immediately after connection
	if err := setupExchanges(conn); err != nil {
		log.Printf("Warning: Failed to setup RabbitMQ exchanges: %v", err)
	}

	return conn, nil
}

func setupExchanges(conn *amqp091.Connection) error {
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	// Declare exchanges
	exchanges := []struct {
		name string
		kind string
	}{
		{"ride_topic", "topic"},
		{"driver_topic", "topic"},
		{"location_fanout", "fanout"},
	}

	for _, exchange := range exchanges {
		err = ch.ExchangeDeclare(
			exchange.name,
			exchange.kind,
			true,  // durable
			false, // auto-deleted
			false, // internal
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to declare exchange %s: %w", exchange.name, err)
		}
	}

	log.Println("RabbitMQ exchanges setup completed")
	return nil
}
