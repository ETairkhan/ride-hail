package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"ride-hail/internal/domain/models"
	"ride-hail/internal/domain/services"
	"ride-hail/internal/common/middleware" // Add this import
)

type RideHandler struct {
	rideService services.RideService
}

func NewRideHandler(rideService services.RideService) *RideHandler {
	return &RideHandler{rideService: rideService}
}

// SetupRoutes sets up all the HTTP routes for ride operations
func (h *RideHandler) SetupRoutes(mux *http.ServeMux) {
	// Note: Routes will be protected by AuthMiddleware in app layer
	mux.HandleFunc("POST /rides", h.CreateRide)
	mux.HandleFunc("POST /rides/{ride_id}/cancel", h.CancelRide)
}

func (h *RideHandler) CreateRide(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get authenticated user from context (set by AuthMiddleware)
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	var req models.CreateRideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Use authenticated user's ID as passenger ID
	req.PassengerID = user.ID

	if req.PickupAddress == "" || req.DestinationAddress == "" {
		http.Error(w, "Pickup and destination addresses are required", http.StatusBadRequest)
		return
	}

	// Validate coordinates
	if !isValidLatitude(req.PickupLatitude) || !isValidLongitude(req.PickupLongitude) {
		http.Error(w, "Invalid pickup coordinates", http.StatusBadRequest)
		return
	}

	if !isValidLatitude(req.DestinationLatitude) || !isValidLongitude(req.DestinationLongitude) {
		http.Error(w, "Invalid destination coordinates", http.StatusBadRequest)
		return
	}

	// Create ride
	response, err := h.rideService.CreateRide(r.Context(), &req)
	if err != nil {
		log.Printf("Error creating ride: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create ride: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func (h *RideHandler) CancelRide(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rideID := r.PathValue("ride_id")
	if rideID == "" {
		http.Error(w, "Ride ID is required", http.StatusBadRequest)
		return
	}

	var req models.CancelRideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	req.RideID = rideID

	if err := h.rideService.CancelRide(r.Context(), &req); err != nil {
		log.Printf("Error cancelling ride: %v", err)
		http.Error(w, fmt.Sprintf("Failed to cancel ride: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"ride_id":      rideID,
		"status":       "CANCELLED",
		"cancelled_at": nil, // This would be set by the service
		"message":      "Ride cancelled successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func isValidLatitude(lat float64) bool {
	return lat >= -90 && lat <= 90
}

func isValidLongitude(lng float64) bool {
	return lng >= -180 && lng <= 180
}
