package models

import "time"

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
