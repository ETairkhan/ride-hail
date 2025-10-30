package ports

import (
	"ride-hail/internal/ride-service/core/domain/data"
	messagebrokerdto "ride-hail/internal/ride-service/core/domain/message_broker_dto"
	websocketdto "ride-hail/internal/ride-service/core/domain/websocket_dto"
)

type IRidesService interface {
	CreateRide(data.RidesRequestDto) (data.RidesResponseDto, error)
	CancelRide(data.RidesCancelRequestDto, string) (data.RideCancelResponseDto, error)

	// input: rideId, driverId, output: passengerId, rideNumber, error
	// set to status match, and also send to the exchange
	SetStatusMatch(string, string) (passengerId string, rideNumber string, err error)
	EstimateDistance(rideId string, longitude, latitude, speed float64) (passengerId, estimatedTime string, distance float64, err error)
	CancelEveryPossibleRides() error
	UpdateRideStatus(messagebrokerdto.DriverStatusUpdate) (string, websocketdto.Event, error)
}

type IPassengerService interface {
	IsPassengerExists(passengerId string) (bool, error)
	// output passengerId
}
