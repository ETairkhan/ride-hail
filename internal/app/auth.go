package app

import (
	"encoding/json"
	"log"
	"net/http"
	"ride-hail/internal/adapter/handlers"
	"ride-hail/internal/common/config"
	"ride-hail/internal/common/middleware"
	"ride-hail/internal/domain/repo"
	"ride-hail/internal/domain/services"

	"github.com/jackc/pgx/v5"
)

// AuthStartService initializes and starts the authentication service
func AuthStartService(cfg config.Config, dbConn *pgx.Conn) {
	// Initialize repositories
	authRepo := repo.NewAuthRepository(dbConn)
	driverRepo := repo.NewDriverRepository(dbConn)

	// Initialize services
	authService := services.NewAuthServiceWithDriver(authRepo, driverRepo, cfg)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)

	// Setup routes
	mux := http.NewServeMux()

	// Public routes (no authentication required)
	authHandler.SetupRoutes(mux)

	// Protected routes with JWT middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.DBConfig.JWTSecret)
	mux.Handle("GET /auth/profile", authMiddleware.Wrap(http.HandlerFunc(authHandler.GetProfile)))

	// Health check endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "service": "auth"})
	})

	// Start server
	log.Printf("Starting Auth Service on port %s", cfg.ServicesConfig.AuthServicePort)
	err := http.ListenAndServe(":"+cfg.ServicesConfig.AuthServicePort, mux)
	if err != nil {
		log.Fatalf("Error starting Auth Service: %v", err)
	}
}
