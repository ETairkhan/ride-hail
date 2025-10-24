package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ride-hail/internal/config"
	"ride-hail/internal/database"
	"ride-hail/internal/handlers"
	"ride-hail/internal/services"
	"ride-hail/websocket"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Log configuration (without passwords)
	log.Printf("Configuration loaded: DB=%s:%d, RabbitMQ=%s:%d, ServicePort=%d",
		cfg.Database.Host, cfg.Database.Port,
		cfg.RabbitMQ.Host, cfg.RabbitMQ.Port,
		cfg.Services.DriverLocationService)

	// Initialize database connection
	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Initialize RabbitMQ connection (with retry logic)
	var rabbitConn *database.RabbitMQ
	rabbitConn, err = database.NewRabbitMQ(cfg.RabbitMQ)
	if err != nil {
		log.Printf("Warning: Failed to connect to RabbitMQ: %v", err)
		log.Println("Application will start without RabbitMQ functionality")
		// Continue without RabbitMQ - set to nil
		rabbitConn = nil
	} else {
		defer rabbitConn.Close()
	}

	// Initialize services
	driverService := services.NewDriverService(db, rabbitConn, wsHub, cfg)

	// Initialize handlers
	driverHandler := handlers.NewDriverHandler(driverService)

	// Start HTTP server
	server := handlers.NewServer(cfg, driverHandler, wsHub)

	// Start server in a goroutine
	go func() {
		log.Printf("Starting Driver & Location Service on port %d", cfg.Services.DriverLocationService)
		if err := server.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
