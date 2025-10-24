package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"ride-hail/internal/config"
	"ride-hail/internal/services"
	"ride-hail/models"
	rideWebsocket "ride-hail/websocket"
)

type DriverHandler struct {
	driverService *services.DriverService
}

// Create a separate upgrader variable to avoid conflicts
var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // In production, validate origins
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func NewDriverHandler(driverService *services.DriverService) *DriverHandler {
	return &DriverHandler{
		driverService: driverService,
	}
}

type Server struct {
	router        *http.ServeMux
	cfg           *config.Config
	driverHandler *DriverHandler
	wsHub         *rideWebsocket.Hub // WebSocket hub for managing connections
	server        *http.Server
}

func NewServer(cfg *config.Config, driverHandler *DriverHandler, wsHub *rideWebsocket.Hub) *Server {
	s := &Server{
		router:        http.NewServeMux(),
		cfg:           cfg,
		driverHandler: driverHandler,
		wsHub:         wsHub,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Health check endpoint
	s.router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})
	// Define routes
	s.router.HandleFunc("/drivers/{driver_id}/online", s.driverHandler.GoOnline)
	s.router.HandleFunc("/drivers/{driver_id}/offline", s.driverHandler.GoOffline)
	s.router.HandleFunc("/drivers/{driver_id}/location", s.driverHandler.UpdateLocation)
	s.router.HandleFunc("/drivers/{driver_id}/start", s.driverHandler.StartRide)
	s.router.HandleFunc("/drivers/{driver_id}/complete", s.driverHandler.CompleteRide)
	s.router.HandleFunc("/ws/drivers/{driver_id}", s.serveDriverWebSocket)
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.Services.DriverLocationService)
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	log.Printf("Starting Driver & Location Service on %s", addr)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *Server) serveDriverWebSocket(w http.ResponseWriter, r *http.Request) {
	driverID := r.URL.Path[len("/ws/drivers/"):] // Extract driverID from URL path

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		http.Error(w, "WebSocket upgrade failed", http.StatusInternalServerError)
		return
	}

	client := &rideWebsocket.Client{ // WebSocket client structure
		ID:       driverID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		UserType: "driver",
	}

	s.wsHub.Register <- client

	// Start goroutines for reading and writing
	go client.WritePump()
	go client.ReadPump()
}

// Response types for API responses
type OnlineResponse struct {
	Status    string `json:"status"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type OfflineResponse struct {
	Status         string                   `json:"status"`
	SessionID      string                   `json:"session_id"`
	SessionSummary *services.SessionSummary `json:"session_summary"`
	Message        string                   `json:"message"`
}

type LocationResponse struct {
	CoordinateID string    `json:"coordinate_id"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type StartRideResponse struct {
	RideID    string    `json:"ride_id"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
	Message   string    `json:"message"`
}

type CompleteRideResponse struct {
	RideID         string    `json:"ride_id"`
	Status         string    `json:"status"`
	CompletedAt    time.Time `json:"completed_at"`
	DriverEarnings float64   `json:"driver_earnings"`
	Message        string    `json:"message"`
}

// DriverHandler methods

func (h *DriverHandler) GoOnline(w http.ResponseWriter, r *http.Request) {
	driverID := r.URL.Path[len("/drivers/"):] // Extract driverID

	var req struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	session, err := h.driverService.GoOnline(ctx, driverID, req.Latitude, req.Longitude)
	if err != nil {
		log.Printf("Error going online: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(session); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func (h *DriverHandler) GoOffline(w http.ResponseWriter, r *http.Request) {
	driverID := r.URL.Path[len("/drivers/"):] // Extract driverID

	ctx := r.Context()
	session, err := h.driverService.GoOffline(ctx, driverID)
	if err != nil {
		log.Printf("Error going offline: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(session); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func (h *DriverHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	driverID := r.URL.Path[len("/drivers/"):] // Extract driverID

	var req models.LocationUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	coord, err := h.driverService.UpdateLocation(ctx, driverID, req)
	if err != nil {
		log.Printf("Error updating location: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := LocationResponse{
		CoordinateID: coord.ID,
		UpdatedAt:    coord.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func (h *DriverHandler) StartRide(w http.ResponseWriter, r *http.Request) {
	driverID := r.URL.Path[len("/drivers/"):] // Extract driverID

	var req struct {
		RideID   string          `json:"ride_id"`
		Location models.Location `json:"driver_location"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	result, err := h.driverService.StartRide(ctx, driverID, req.RideID, req.Location)
	if err != nil {
		log.Printf("Error starting ride: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := StartRideResponse{
		RideID:    result["ride_id"].(string),
		Status:    result["status"].(string),
		StartedAt: result["started_at"].(time.Time),
		Message:   result["message"].(string),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func (h *DriverHandler) CompleteRide(w http.ResponseWriter, r *http.Request) {
	driverID := r.URL.Path[len("/drivers/"):] // Extract driverID

	var req struct {
		RideID                string          `json:"ride_id"`
		FinalLocation         models.Location `json:"final_location"`
		ActualDistanceKM      float64         `json:"actual_distance_km"`
		ActualDurationMinutes int             `json:"actual_duration_minutes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	result, err := h.driverService.CompleteRide(ctx, driverID, req.RideID, req.FinalLocation, req.ActualDistanceKM, req.ActualDurationMinutes)
	if err != nil {
		log.Printf("Error completing ride: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := CompleteRideResponse{
		RideID:         result["ride_id"].(string),
		Status:         result["status"].(string),
		CompletedAt:    result["completed_at"].(time.Time),
		DriverEarnings: result["driver_earnings"].(float64),
		Message:        result["message"].(string),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}
