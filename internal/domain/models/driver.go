package models

import (
	"time"
)

type Driver struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Email         string    `json:"email"`
	LicenseNumber string    `json:"license_number"`
	VehicleMake   string    `json:"vehicle_make"`
	VehicleModel  string    `json:"vehicle_model"`
	VehicleYear   int       `json:"vehicle_year"`
	VehicleColor  string    `json:"vehicle_color"`
	LicensePlate  string    `json:"license_plate"`
	VehicleType   string    `json:"vehicle_type"`
	Status        string    `json:"status"`
	Rating        float64   `json:"rating"`
	TotalRides    int       `json:"total_rides"`
	TotalEarnings float64   `json:"total_earnings"`
	IsVerified    bool      `json:"is_verified"`
	CurrentRideID string    `json:"current_ride_id,omitempty"`
	Latitude      float64   `json:"latitude,omitempty"`
	Longitude     float64   `json:"longitude,omitempty"`
	DistanceKM    float64   `json:"distance_km,omitempty"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type LocationWithAddress struct {
	Location
	Address string `json:"address"`
}

type LocationUpdate struct {
	Location
	Address        string  `json:"address"`
	AccuracyMeters float64 `json:"accuracy_meters,omitempty"`
	SpeedKMH       float64 `json:"speed_kmh,omitempty"`
	HeadingDegrees float64 `json:"heading_degrees,omitempty"`
}

type LocationMessage struct {
	DriverID       string    `json:"driver_id"`
	RideID         string    `json:"ride_id,omitempty"`
	Location       Location  `json:"location"`
	SpeedKMH       float64   `json:"speed_kmh,omitempty"`
	HeadingDegrees float64   `json:"heading_degrees,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// Update the RideRequest struct to use VehicleType instead of RideType
type RideRequest struct {
	RideID                   string   `json:"ride_id"`
	RideNumber               string   `json:"ride_number"`
	PassengerID              string   `json:"passenger_id"`
	VehicleType              string   `json:"vehicle_type"`
	PickupLocation           Location `json:"pickup_location"`
	PickupAddress            string   `json:"pickup_address"`
	DestinationLocation      Location `json:"destination_location"`
	DestinationAddress       string   `json:"destination_address"`
	EstimatedFare            float64  `json:"estimated_fare"`
	EstimatedDurationMinutes int      `json:"estimated_duration_minutes"`
	MaxDistanceKM            float64  `json:"max_distance_km"`
	TimeoutSeconds           int      `json:"timeout_seconds"`
}

type RideOffer struct {
	OfferID                      string              `json:"offer_id"`
	RideID                       string              `json:"ride_id"`
	RideNumber                   string              `json:"ride_number"`
	PickupLocation               LocationWithAddress `json:"pickup_location"`
	DestinationLocation          LocationWithAddress `json:"destination_location"`
	EstimatedFare                float64             `json:"estimated_fare"`
	DriverEarnings               float64             `json:"driver_earnings"`
	DistanceToPickupKM           float64             `json:"distance_to_pickup_km"`
	EstimatedRideDurationMinutes int                 `json:"estimated_ride_duration_minutes"`
	ExpiresAt                    time.Time           `json:"expires_at"`
}

type DriverStatusMessage struct {
	DriverID  string    `json:"driver_id"`
	Status    string    `json:"status"`
	RideID    string    `json:"ride_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type RideStatusMessage struct {
	RideID    string    `json:"ride_id"`
	Status    string    `json:"status"`
	DriverID  string    `json:"driver_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type JSONMap map[string]interface{}

// WebSocket message types
const (
	MessageTypeLocationUpdate = "location_update"
	MessageTypeRideOffer      = "ride_offer"
	MessageTypeRideStatus     = "ride_status"
	MessageTypeDriverStatus   = "driver_status"
	MessageTypePing           = "ping"
	MessageTypePong           = "pong"
)
