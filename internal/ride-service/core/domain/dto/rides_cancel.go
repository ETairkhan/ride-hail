package dto

import "time"

type RidesCancelRequestDto struct {
	Reason string `json:"reason"`
}

type RidesCancelResponseDto struct {
	RideID      string    `json:"ride_id"`
	Status      string    `json:"status"`
	CancelledAt time.Time `json:"cancelled_at"`
	Message     string    `json:"message"`
}
