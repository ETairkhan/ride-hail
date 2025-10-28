package models

import "time"

type RideStatus string
type VehicleType string

const (
	StatusRequested  RideStatus = "REQUESTED"
	StatusMatched    RideStatus = "MATCHED"
	StatusEnRoute    RideStatus = "EN_ROUTE"
	StatusArrived    RideStatus = "ARRIVED"
	StatusInProgress RideStatus = "IN_PROGRESS"
	StatusCompleted  RideStatus = "COMPLETED"
	StatusCancelled  RideStatus = "CANCELLED"

	VehicleEconomy VehicleType = "ECONOMY"
	VehiclePremium VehicleType = "PREMIUM"
	VehicleXL      VehicleType = "XL"
)

type Ride struct {
	ID                      string      `json:"id"`
	CreatedAt               time.Time   `json:"created_at"`
	UpdatedAt               time.Time   `json:"updated_at"`
	RideNumber              string      `json:"ride_number"`
	PassengerID             string      `json:"passenger_id"`
	DriverID                *string     `json:"driver_id,omitempty"`
	VehicleType             VehicleType `json:"vehicle_type"`
	Status                  RideStatus  `json:"status"`
	Priority                int64       `json:"priority"`
	RequestedAt             time.Time   `json:"requested_at"`
	MatchedAt               *time.Time  `json:"matched_at,omitempty"`
	StartedAt               *time.Time  `json:"started_at,omitempty"`
	CompletedAt             *time.Time  `json:"completed_at,omitempty"`
	CancelledAt             *time.Time  `json:"cancelled_at,omitempty"`
	CancellationReason      string      `json:"cancellation_reason,omitempty"`
	EstimatedFare           float64     `json:"estimated_fare"`
	FinalFare               float64     `json:"final_fare,omitempty"`
	PickupCoordinateID      string      `json:"pickup_coordinate_id"`
	DestinationCoordinateID string      `json:"destination_coordinate_id"`
}

type CreateRideRequest struct {
	PassengerID          string      `json:"passenger_id"`
	PickupLatitude       float64     `json:"pickup_latitude"`
	PickupLongitude      float64     `json:"pickup_longitude"`
	PickupAddress        string      `json:"pickup_address"`
	DestinationLatitude  float64     `json:"destination_latitude"`
	DestinationLongitude float64     `json:"destination_longitude"`
	DestinationAddress   string      `json:"destination_address"`
	VehicleType          VehicleType `json:"ride_type"`
}

type CancelRideRequest struct {
	RideID string `json:"ride_id"`
	Reason string `json:"reason"`
}

type RideResponse struct {
	RideID                   string     `json:"ride_id"`
	RideNumber               string     `json:"ride_number"`
	Status                   RideStatus `json:"status"`
	EstimatedFare            float64    `json:"estimated_fare"`
	EstimatedDurationMinutes int        `json:"estimated_duration_minutes"`
	EstimatedDistanceKm      float64    `json:"estimated_distance_km"`
}
