package types

import (
	"context"
	"time"

	"ride-hail/internal/domain/models"
)

// Response types for API responses
type OnlineResponse struct {
	Status    string `json:"status"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type SessionSummary struct {
	DurationHours  float64 `json:"duration_hours"`
	RidesCompleted int     `json:"rides_completed"`
	Earnings       float64 `json:"earnings"`
}

type OfflineResponse struct {
	Status         string         `json:"status"`
	SessionID      string         `json:"session_id"`
	SessionSummary SessionSummary `json:"session_summary"`
	Message        string         `json:"message"`
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

// Service interfaces to break import cycles
type DriverService interface {
	GoOnline(ctx context.Context, driverID string, lat, lng float64) (*OnlineResponse, error)
	GoOffline(ctx context.Context, driverID string) (*OfflineResponse, error)
	UpdateLocation(ctx context.Context, driverID string, update models.LocationUpdate) (*models.Coordinates, error)
	StartRide(ctx context.Context, driverID, rideID string, location models.Location) (map[string]interface{}, error)
	CompleteRide(ctx context.Context, driverID, rideID string, location models.Location, actualDistanceKM float64, actualDurationMinutes int) (map[string]interface{}, error)
}
