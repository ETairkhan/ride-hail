// internal/adapter/rabbitmq/connection.go
package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ride-hail/internal/common/config"
	"ride-hail/internal/domain/models"

	"github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	Conn *amqp091.Connection
}

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


func SetupExchangesAndQueues(conn *amqp091.Connection) error {
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	// Declare exchanges
	exchanges := map[string]string{
		"ride_topic":     "topic",
		"driver_topic":   "topic",
		"location_fanout": "fanout",
	}

	for name, kind := range exchanges {
		err := ch.ExchangeDeclare(
			name,  // name
			kind,  // type
			true,  // durable
			false, // auto-deleted
			false, // internal
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to declare exchange %s: %w", name, err)
		}
	}

	// Declare and bind queues
	queues := []struct {
		name       string
		exchange   string
		routingKey string
	}{
		{"ride_requests", "ride_topic", "ride.request.*"},
		{"ride_status", "ride_topic", "ride.status.*"},
		{"driver_matching", "ride_topic", "ride.request.*"},
		{"driver_responses", "driver_topic", "driver.response.*"},
		{"driver_status", "driver_topic", "driver.status.*"},
		{"location_updates_ride", "location_fanout", ""},
	}

	for _, q := range queues {
		_, err := ch.QueueDeclare(
			q.name, // name
			true,   // durable
			false,  // delete when unused
			false,  // exclusive
			false,  // no-wait
			nil,    // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", q.name, err)
		}

		if q.exchange != "" {
			err = ch.QueueBind(
				q.name,       // queue name
				q.routingKey, // routing key
				q.exchange,   // exchange
				false,        // no-wait
				nil,          // arguments
			)
			if err != nil {
				return fmt.Errorf("failed to bind queue %s: %w", q.name, err)
			}
		}
	}

	log.Println("RabbitMQ exchanges and queues setup completed")
	return nil
}

func PublishRideRequest(conn *amqp091.Connection, ride *models.Ride, pickupCoords, destCoords *models.Coordinates) error {
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	message := models.RideRequest{
		RideID:     ride.ID,
		RideNumber: ride.RideNumber,
		PickupLocation: models.Location{
			Latitude:  pickupCoords.Latitude,
			Longitude: pickupCoords.Longitude,
		},
		PickupAddress: pickupCoords.Address,
		DestinationLocation: models.Location{
			Latitude:  destCoords.Latitude,
			Longitude: destCoords.Longitude,
		},
		DestinationAddress: destCoords.Address,
		VehicleType:        string(ride.VehicleType),
		EstimatedFare:      ride.EstimatedFare,
		MaxDistanceKM:      5.0,
		TimeoutSeconds:     30,
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	routingKey := fmt.Sprintf("ride.request.%s", ride.VehicleType)

	err = ch.PublishWithContext(context.Background(),
		"ride_topic", // exchange
		routingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
			DeliveryMode: amqp091.Persistent,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Published ride request for ride %s to %s", ride.ID, routingKey)
	return nil
}