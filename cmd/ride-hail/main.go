package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"ride-hail/internal/adapter/db"
	"ride-hail/internal/adapter/handlers"
	"ride-hail/internal/adapter/rabbitmq"
	"ride-hail/internal/adapter/websocket"
	"ride-hail/internal/app"
	"ride-hail/internal/common/config"
	"ride-hail/internal/common/logger"
	"strings"
	"syscall"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

var (
	ErrModeFlag       = errors.New("mode flag is required")
	ErrUnknownService = errors.New("unknown service specified")
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()
	fmt.Printf("Loaded configuration: %+v\n", cfg)

	logger, err := logger.New("INFO")
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	logger.Action("ride_hail_system_started").Info("Ride Hail System starting up")

	// Global flags for selecting the service mode
	fs := flag.NewFlagSet("main", flag.ExitOnError)
	mode := fs.String("mode", "", "service to run: ride-service | driver-service | admin-service | auth-service")

	// Only parse the first few args for `--mode`, the rest go to the service
	args := os.Args[1:]
	modeArgs := []string{}
	for i, arg := range args {
		if strings.HasPrefix(arg, "--mode") || strings.HasPrefix(arg, "-mode") {
			modeArgs = args[:i+1]
			break
		}
	}

	// parse mode
	if err := fs.Parse(modeArgs); err != nil {
		logger.Action("ride_hail_system_failed").Error("Failed to parse flags", err)
		help(fs)
		return
	}

	if *mode == "" {
		logger.Action("ride_hail_system_failed").Error("Failed to start ride hail system", ErrModeFlag)
		help(fs)
		return
	}

	ctx, cancelMain := context.WithCancel(context.Background())

	// Initialize database connection
	dbConnection, err := db.InitDB(cfg.DBConfig, ctx)
	if err != nil {
		log.Fatalf("Error connecting to DB: %v", err)
	}
	defer dbConnection.Close(ctx)

	// Initialize RabbitMQ with retry logic
	var rabbitConn *amqp091.Connection
	rabbitConn, err = rabbitmq.InitRabbitMQ(cfg.RabbitMQConfig)
	if err != nil {
		log.Printf("Warning: Failed to connect to RabbitMQ: %v", err)
		log.Println("Application will start without RabbitMQ functionality")
		rabbitConn = nil
	} else {
		defer rabbitConn.Close()
	}

	// Create a channel to wait for server shutdown
	done := make(chan bool, 1)

	switch *mode {
	case "admin-service", "as":
		l := logger.With("service", "admin-service")
		l.Action("admin_service_started").Info("Admin Service starting up")

		// TODO: Initialize admin service components
		// adminService := app.NewAdminService(dbConnection, rabbitConn)
		// adminHandler := handlers.NewAdminHandler(adminService)

		// Start admin service HTTP server
		// go app.startAdminService(cfg, l, done)

		l.Action("admin_service_started").Info("Admin Service started successfully")

	case "driver-location-service", "dls":
		driverServiceLogger := logger.With("service", "driver-location-service")
		driverServiceLogger.Action("driver_location_service_started").Info("Driver and Location service starting up")

		// Initialize WebSocket hub
		wsHub := websocket.NewHub()
		go wsHub.Run()

		// Initialize services
		driverService := app.NewDriverService(dbConnection, &rabbitmq.RabbitMQ{Conn: rabbitConn}, wsHub, &cfg)

		// Initialize handlers
		driverHandler := handlers.NewDriverHandler(driverService)

		// Start HTTP server
		server := handlers.NewServer(&cfg, driverHandler, wsHub)

		// Start server in a goroutine
		go func() {
			driverServiceLogger.Action("driver_location_service_starting").Info("Starting Driver & Location Service")
			if err := server.Start(); err != nil && err != http.ErrServerClosed {
				driverServiceLogger.Action("driver_location_service_failed").Error("Failed to start server", err)
				done <- true
			}
		}()

		driverServiceLogger.Action("driver_location_service_started").Info("Driver and Location service started successfully")

	case "ride-service", "rs":
		l := logger.With("service", "ride-service")
		l.Action("ride_service_started").Info("Ride Service starting up")

		// Start ride service
		go func() {
			app.RideStartService(cfg, dbConnection, rabbitConn)
			done <- true
		}()

		l.Action("ride_service_started").Info("Ride Service started successfully")

	case "auth-service", "au":
		l := logger.With("service", "auth-service")
		l.Action("auth_service_started").Info("Auth Service starting up")

		// For now, auth is part of ride service
		// You can separate it later if needed
		l.Action("auth_service_info").Info("Auth functionality is included in ride-service")
		done <- true

	default:
		logger.Action("ride_hail_system_failed").Error("Failed to start ride hail system", ErrUnknownService)
		help(fs)
		return
	}

	// Wait for interrupt signal or service completion
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-stop:
		logger.Action("service_shutdown").Info("Received shutdown signal")
	case <-done:
		logger.Action("service_completed").Info("Service completed execution")
	}

	logger.Action("service_cleanup").Info("Shutting down services...")

	// Give services time to shut down gracefully
	_, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Cancel main context
	cancelMain()

	// TODO: Add graceful shutdown for HTTP servers
	logger.Action("service_exit").Info("Service exiting")
}

func help(fs *flag.FlagSet) {
	fmt.Println("\n🚗 Ride Hail System - Usage:")
	fmt.Println("  go run cmd/main.go --mode=<service>")
	fmt.Println("\nAvailable Services:")
	fmt.Println("  ride-service (rs)           - Orchestrates ride lifecycle and passenger interactions")
	fmt.Println("  driver-service (dls)        - Handles driver operations, matching, and location tracking")
	fmt.Println("  admin-service (as)          - Provides monitoring, analytics, and system oversight")
	fmt.Println("  auth-service (au)           - User authentication and authorization")
	fmt.Println("\nExamples:")
	fmt.Println("  go run cmd/main.go --mode=ride-service")
	fmt.Println("  go run cmd/main.go --mode=driver-service")
	fmt.Println("  go run cmd/main.go --mode=admin-service")
	fmt.Println("  go run cmd/main.go --mode=auth-service")
	fmt.Println("\nShort Forms:")
	fmt.Println("  rs  - ride-service")
	fmt.Println("  dls - driver-location-service")
	fmt.Println("  as  - admin-service")
	fmt.Println("  au  - auth-service")
	fmt.Println("\nConfiguration:")
	fmt.Println("  Use environment variables or config files for database, RabbitMQ, and service settings")
}
