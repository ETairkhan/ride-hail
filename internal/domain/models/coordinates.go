package models

import "time"

type Coordinates struct {
	ID              string    `json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	EntityID        string    `json:"entity_id"` // driver_id or passenger_id
	EntityType      string    `json:"entity_type"` // driver or passenger
	Address         string    `json:"address"`
	Latitude        float64   `json:"latitude"`
	Longitude       float64   `json:"longitude"`
	FareAmount      float64   `json:"fare_amount,omitempty"`
	DistanceKm      float64   `json:"distance_km,omitempty"`
	DurationMinutes int64     `json:"duration_minutes,omitempty"`
	IsCurrent       bool      `json:"is_current"`
}