// app/ride.go
package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"ride-hail/internal/adapter/handlers"
	"ride-hail/internal/adapter/rabbitmq"
	"ride-hail/internal/common/config"
	"ride-hail/internal/common/middleware"
	"ride-hail/internal/domain/models"
	"ride-hail/internal/domain/repo"
	"ride-hail/internal/domain/services"

	"github.com/jackc/pgx/v5"
	"github.com/rabbitmq/amqp091-go"
)

type RabbitMQPublisher struct {
	rabbitMQ *rabbitmq.RabbitMQ
}

func NewRabbitMQPublisher(rabbitMQ *rabbitmq.RabbitMQ) *RabbitMQPublisher {
	return &RabbitMQPublisher{rabbitMQ: rabbitMQ}
}

func (p *RabbitMQPublisher) PublishRideRequest(ctx context.Context, ride *models.Ride, pickupCoords, destCoords *models.Coordinates) error {
	// Use the centralized RabbitMQ function
	return rabbitmq.PublishRideRequest(p.rabbitMQ.Conn, ride, pickupCoords, destCoords)
}

func RideStartService(config config.Config, dbConn *pgx.Conn, rabbitConn *amqp091.Connection) {
	// Initialize repositories
	authRepo := repo.NewAuthRepository(dbConn)
	driverRepo := repo.NewDriverRepository(dbConn)
	rideRepo := repo.NewRideRepository(dbConn)

	// Initialize services - create RabbitMQ wrapper
	var publisher services.MessagePublisher
	if rabbitConn != nil {
		rabbitMQ := &rabbitmq.RabbitMQ{Conn: rabbitConn}
		publisher = NewRabbitMQPublisher(rabbitMQ)
	} else {
		log.Println("RabbitMQ not available - using mock publisher")
		publisher = &mockPublisher{}
	}

	authService := services.NewAuthServiceWithDriver(authRepo, driverRepo, config)
	rideService := services.NewRideService(rideRepo, authRepo, publisher)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	rideHandler := handlers.NewRideHandler(rideService)

	// Setup routes
	mux := http.NewServeMux()

	// Public routes (no authentication required)
	authHandler.SetupRoutes(mux)

	// Protected routes with your JWT middleware
	authMiddleware := middleware.NewAuthMiddleware(config.DBConfig.JWTSecret)
	mux.Handle("GET /auth/profile", authMiddleware.Wrap(http.HandlerFunc(authHandler.GetProfile)))
	mux.Handle("POST /rides", authMiddleware.Wrap(http.HandlerFunc(rideHandler.CreateRide)))
	mux.Handle("POST /rides/{ride_id}/cancel", authMiddleware.Wrap(http.HandlerFunc(rideHandler.CancelRide)))

	// Start server
	log.Printf("Starting Ride Service on port %s", config.ServicesConfig.RideServicePort)
	err := http.ListenAndServe(":"+config.ServicesConfig.RideServicePort, mux)
	if err != nil {
		log.Fatalf("Error starting Ride Service: %v", err)
	}
}

// Mock publisher for when RabbitMQ is not available
type mockPublisher struct{}

func (m *mockPublisher) PublishRideRequest(ctx context.Context, ride *models.Ride, pickupCoords, destCoords *models.Coordinates) error {
	log.Printf("Mock: Would publish ride request for ride %s", ride.ID)
	return nil
}