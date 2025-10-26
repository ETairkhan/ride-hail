package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ride-hail/internal/adapter/db"
	"ride-hail/internal/adapter/rabbitmq"
	"ride-hail/internal/app"
	"ride-hail/internal/common/config"

	"ride-hail/internal/adapter/handlers"
	"ride-hail/internal/adapter/websocket"
)

func main() {
	// Load configuration
	ctx := context.Background()
	cfg := config.LoadConfig()
	fmt.Println(cfg)

	// Initialize database connection
	dbConnection, err := db.InitDB(cfg.DBConfig, ctx)
	if err != nil {
		log.Fatalf("Error connecting to DB: %v", err)
	}
	defer dbConnection.Close(ctx)

	// Initialize RabbitMQ with retry logic
	var rabbitConnection *rabbitmq.RabbitMQ
	rabbitConn, err := rabbitmq.InitRabbitMQ(cfg.RabbitMQConfig)
	if err != nil {
		log.Printf("Warning: Failed to connect to RabbitMQ: %v", err)
		log.Println("Application will start without RabbitMQ functionality")
		rabbitConnection = nil
	} else {
		rabbitConnection = &rabbitmq.RabbitMQ{Conn: rabbitConn}
		defer rabbitConnection.Conn.Close()
	}

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Initialize services
	driverService := app.NewDriverService(dbConnection, rabbitConnection, wsHub, &cfg)

	// Initialize handlers
	driverHandler := handlers.NewDriverHandler(driverService)

	// Start HTTP server
	server := handlers.NewServer(&cfg, driverHandler, wsHub)

	// Start server in a goroutine
	go func() {
		log.Printf("Starting Driver & Location Service on port %s", cfg.ServicesConfig.DriverLocationServicePort)
		if err := server.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down services...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
