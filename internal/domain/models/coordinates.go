package models

import "time"

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
