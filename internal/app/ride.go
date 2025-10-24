package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"ride-hail/config"
	"ride-hail/internal/common/middleware"
	"ride-hail/internal/domain/repo"
	"ride-hail/internal/domain/services"
	"ride-hail/internal/domain/models"
	"ride-hail/internal/adapter/handlers"

	"github.com/jackc/pgx/v5"
	"github.com/rabbitmq/amqp091-go"
)

type RabbitMQPublisher struct {
	conn *amqp091.Connection
}

func NewRabbitMQPublisher(conn *amqp091.Connection) *RabbitMQPublisher {
	return &RabbitMQPublisher{conn: conn}
}

func (p *RabbitMQPublisher) PublishRideRequest(ctx context.Context, ride *models.Ride, pickupCoords, destCoords *models.Coordinates) error {
	ch, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	message := map[string]interface{}{
		"ride_id": ride.ID,
		"ride_number": ride.RideNumber,
		"pickup_location": map[string]interface{}{
			"lat":     pickupCoords.Latitude,
			"lng":     pickupCoords.Longitude,
			"address": pickupCoords.Address,
		},
		"destination_location": map[string]interface{}{
			"lat":     destCoords.Latitude,
			"lng":     destCoords.Longitude,
			"address": destCoords.Address,
		},
		"ride_type":       ride.VehicleType,
		"estimated_fare":  ride.EstimatedFare,
		"max_distance_km": 5.0,
		"timeout_seconds": 30,
	}

	messageBody, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	routingKey := fmt.Sprintf("ride.request.%s", ride.VehicleType)
	
	err = ch.Publish(
		"ride_topic",   // exchange
		routingKey,     // routing key
		false,          // mandatory
		false,          // immediate
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        messageBody,
		},
	)
	
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}
	
	return nil
}

func RideStartService(config config.Config, dbConn *pgx.Conn, rabbitConn *amqp091.Connection) {
	// Initialize repositories
	authRepo := repo.NewAuthRepository(dbConn)
	rideRepo := repo.NewRideRepository(dbConn)
	
	// Initialize services
	authService := services.NewAuthService(authRepo, config)
	rideService := services.NewRideService(rideRepo, authRepo, NewRabbitMQPublisher(rabbitConn))
	
	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	rideHandler := handlers.NewRideHandler(rideService)
	
	// Setup routes
	mux := http.NewServeMux()
	
	// Public routes (no authentication required)
	authHandler.SetupRoutes(mux)
	
	// Protected routes with your JWT middleware
	authMiddleware := middleware.NewAuthMiddleware(config.JWTSecret)
	mux.Handle("GET /auth/profile", authMiddleware.Wrap(http.HandlerFunc(authHandler.GetProfile)))
	mux.Handle("POST /rides", authMiddleware.Wrap(http.HandlerFunc(rideHandler.CreateRide)))
	mux.Handle("POST /rides/{ride_id}/cancel", authMiddleware.Wrap(http.HandlerFunc(rideHandler.CancelRide)))
	
	
	// Start server
	log.Printf("Starting Ride Service on port %s", config.RideServicePort)
	err := http.ListenAndServe(":"+config.RideServicePort, mux)
	if err != nil {
		log.Fatalf("Error starting Ride Service: %v", err)
	}
}