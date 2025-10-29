package operator

import (
	"context"
	"ride-hail/internal/driver-location-service/core/domain/data"
)

type IDriverService interface {
	GoOnline(ctx context.Context, coord data.DriverCoordinatesDTO) (data.DriverOnlineResponse, error)
	GoOffline(ctx context.Context, driver_id string) (data.DriverOfflineRespones, error)
	UpdateLocation(ctx context.Context, request data.NewLocation, driver_id string) (data.NewLocationResponse, error)
	StartRide(ctx context.Context, requestMessage data.StartRide) (data.StartRideResponse, error)
	CompleteRide(ctx context.Context, request data.RideCompleteForm) (data.RideCompleteResponse, error)
}
