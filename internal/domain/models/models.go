package models

import "time"

// User struct
type User struct {
	UserID       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Email        string
	Role         string // PASSENGER, DRIVER, ADMIN
	Status       string // ACTIVE, INACTIVE, BANNED
	PasswordHash string
}

// Coordinates struct
type Coordinates struct {
	CoordinateID    string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	EntityID        string // driver_id or passenger_id
	EntityType      string // driver or passenger
	Address         string
	Latitude        float64
	Longitude       float64
	FareAmount      float64
	DistanceKm      float64
	DurationMinutes int64
	IsCurrent       bool
}

// Rides struct now includes pickup and destination coordinates
type Rides struct {
	RideID                 string
	CreatedAt              time.Time
	UpdatedAt              time.Time
	RideNumber             string
	PassengerID            string
	DriverID               string
	VehicleType            string // ECONOMY, PREMIUM, XL
	Status                 string // REQUESTED, MATCHED, EN_ROUTE, ARRIVED, IN_PROGRESS, COMPLETED, CANCELLED
	Priority               int64
	RequestedAt            time.Time
	MatchedAt              time.Time
	StartedAt              time.Time
	CompletedAt            time.Time
	CancelledAt            time.Time
	CancellationReason     string
	EstimatedFare          float64
	FinalFare              float64
	PickupCoordinates      Coordinates
	DestinationCoordinates Coordinates
}
