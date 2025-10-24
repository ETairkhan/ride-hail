package repo

import (
	"context"
	"ride-hail/internal/domain/models"

	"github.com/jackc/pgx/v5"
)

type RideRepository interface {
	CreateRide(ctx context.Context, ride *models.Ride) error
	GetRideByID(ctx context.Context, rideID string) (*models.Ride, error)
	UpdateRideStatus(ctx context.Context, rideID string, status models.RideStatus) error
	CancelRide(ctx context.Context, rideID string, reason string) error
	CreateCoordinates(ctx context.Context, coord *models.Coordinates) error
}

type rideRepository struct {
	db *pgx.Conn
}

func NewRideRepository(db *pgx.Conn) RideRepository {
	return &rideRepository{db: db}
}

func (r *rideRepository) CreateRide(ctx context.Context, ride *models.Ride) error {
	query := `
		INSERT INTO rides (
			id, ride_number, passenger_id, vehicle_type, status, priority,
			estimated_fare, pickup_coordinate_id, destination_coordinate_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	
	_, err := r.db.Exec(ctx, query,
		ride.ID,
		ride.RideNumber,
		ride.PassengerID,
		ride.VehicleType,
		ride.Status,
		ride.Priority,
		ride.EstimatedFare,
		ride.PickupCoordinateID,
		ride.DestinationCoordinateID,
	)
	return err
}

func (r *rideRepository) GetRideByID(ctx context.Context, rideID string) (*models.Ride, error) {
	query := `
		SELECT id, created_at, updated_at, ride_number, passenger_id, driver_id,
			   vehicle_type, status, priority, requested_at, matched_at, started_at,
			   completed_at, cancelled_at, cancellation_reason, estimated_fare,
			   final_fare, pickup_coordinate_id, destination_coordinate_id
		FROM rides WHERE id = $1
	`
	
	var ride models.Ride
	err := r.db.QueryRow(ctx, query, rideID).Scan(
		&ride.ID, &ride.CreatedAt, &ride.UpdatedAt, &ride.RideNumber,
		&ride.PassengerID, &ride.DriverID, &ride.VehicleType, &ride.Status,
		&ride.Priority, &ride.RequestedAt, &ride.MatchedAt, &ride.StartedAt,
		&ride.CompletedAt, &ride.CancelledAt, &ride.CancellationReason,
		&ride.EstimatedFare, &ride.FinalFare, &ride.PickupCoordinateID,
		&ride.DestinationCoordinateID,
	)
	
	if err != nil {
		return nil, err
	}
	return &ride, nil
}

func (r *rideRepository) UpdateRideStatus(ctx context.Context, rideID string, status models.RideStatus) error {
	query := `UPDATE rides SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, status, rideID)
	return err
}

func (r *rideRepository) CancelRide(ctx context.Context, rideID string, reason string) error {
	query := `
		UPDATE rides 
		SET status = 'CANCELLED', cancellation_reason = $1, cancelled_at = NOW(), updated_at = NOW()
		WHERE id = $2
	`
	_, err := r.db.Exec(ctx, query, reason, rideID)
	return err
}

func (r *rideRepository) CreateCoordinates(ctx context.Context, coord *models.Coordinates) error {
	query := `
		INSERT INTO coordinates (
			id, entity_id, entity_type, address, latitude, longitude,
			fare_amount, distance_km, duration_minutes, is_current
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	
	_, err := r.db.Exec(ctx, query,
		coord.ID,
		coord.EntityID,
		coord.EntityType,
		coord.Address,
		coord.Latitude,
		coord.Longitude,
		coord.FareAmount,
		coord.DistanceKm,
		coord.DurationMinutes,
		coord.IsCurrent,
	)
	return err
}