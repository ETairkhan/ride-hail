package database

import (
	"context"
	"fmt"
	"ride-hail/internal/ride-service/core/domain/dto"
	"ride-hail/internal/ride-service/core/domain/model"
	"ride-hail/internal/ride-service/core/ports"
	"time"

	"github.com/jackc/pgx/v5"
)

type RidesRepo struct {
	db *DB
}

func NewRidesRepo(db *DB) ports.IRidesRepo {
	return &RidesRepo{
		db: db,
	}
}

func (rr *RidesRepo) GetDistance(ctx context.Context, req dto.RidesRequestDto) (float64, error) {
	q := `SELECT ST_Distance(ST_MakePoint($1, $2)::geography, ST_MakePoint($3, $4)::geography) / 1000 as distance_km`

	db := rr.db.conn
	row := db.QueryRow(ctx, q, req.PickUpLongitude, req.PickUpLatitude, req.DestinationLongitude, req.DestinationLatitude)
	distance := 0.0
	err := row.Scan(&distance)
	if err != nil {
		return 0.0, err
	}
	return distance, nil
}

func (rr *RidesRepo) GetNumberRides(ctx context.Context) (int64, error) {
	q := `
	SELECT 
		COUNT(*) 
	FROM 
		rides
	WHERE
		created_at::date = current_date
	`
	db := rr.db.conn
	row := db.QueryRow(ctx, q)
	var count int64 = 0
	err := row.Scan(&count)
	if err != nil {
		return 0.0, err
	}
	return count, nil
}

func (rr *RidesRepo) CreateRide(ctx context.Context, m model.Rides) (string, error) {
	conn := rr.db.conn
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	// pick up coordinates
	q1 := `INSERT INTO coordinates(
			entity_id, 
			entity_type,
			address, 
			latitude, 
			longitude, 
			fare_amount, 
			distance_km, 
			duration_minutes, 
			is_current
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING coord_id`

	row := tx.QueryRow(ctx, q1,
		m.PassengerId,
		m.PickupCoordinate.EntityType,
		m.PickupCoordinate.Address,
		m.PickupCoordinate.Latitude,
		m.PickupCoordinate.Longitude,
		m.PickupCoordinate.FareAmount,
		m.PickupCoordinate.DistanceKm,
		m.PickupCoordinate.DurationMinutes,
		m.PickupCoordinate.IsCurrent,
	)
	PickupCoordinateId := ""
	if err := row.Scan(&PickupCoordinateId); err != nil {
		tx.Rollback(ctx)
		return "", err
	}
	// destination coordinates
	q2 := `INSERT INTO coordinates(
			entity_id, 
			entity_type,
			address, 
			latitude, 
			longitude, 
			fare_amount, 
			distance_km, 
			duration_minutes, 
			is_current
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING coord_id`

	row = tx.QueryRow(ctx, q2,
		m.PassengerId,
		m.DestinationCoordinate.EntityType,
		m.DestinationCoordinate.Address,
		m.DestinationCoordinate.Latitude,
		m.DestinationCoordinate.Longitude,
		m.DestinationCoordinate.FareAmount,
		m.DestinationCoordinate.DistanceKm,
		m.DestinationCoordinate.DurationMinutes,
		m.DestinationCoordinate.IsCurrent,
	)
	DestinationCoordinateId := ""
	if err := row.Scan(&DestinationCoordinateId); err != nil {
		tx.Rollback(ctx)
		return "", err
	}
	// rides
	q3 := `INSERT INTO rides(
		ride_number,
		passenger_id,
		status,
		priority, 
		estimated_fare,
		final_fare, 
		pickup_coord_id, 
		destination_coord_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING ride_id`

	row = tx.QueryRow(ctx, q3,
		m.RideNumber,
		m.PassengerId,
		m.Status,
		m.Priority,
		m.EstimatedFare,
		m.FinalFare,
		PickupCoordinateId,
		DestinationCoordinateId,
	)

	RideId := ""
	if err := row.Scan(&RideId); err != nil {
		tx.Rollback(ctx)
		return "", err
	}

	return RideId, tx.Commit(ctx)
}

func (rr *RidesRepo) GetRideByID(ctx context.Context, rideID string) (model.Rides, error) {
    q := `
    SELECT 
        r.ride_id,
        r.created_at,
        r.updated_at,
        r.ride_number,
        r.passenger_id,
        r.driver_id,
        r.vehicle_type,
        r.status,
        r.priority,
        r.requested_at,
        r.matched_at,
        r.arrived_at,
        r.started_at,
        r.completed_at,
        r.cancelled_at,
        r.cancellation_reason,
        r.estimated_fare,
        r.final_fare,
        pickup.coord_id as pickup_coord_id,
        pickup.created_at as pickup_created_at,
        pickup.updated_at as pickup_updated_at,
        pickup.entity_id as pickup_entity_id,
        pickup.entity_type as pickup_entity_type,
        pickup.address as pickup_address,
        pickup.latitude as pickup_latitude,
        pickup.longitude as pickup_longitude,
        pickup.fare_amount as pickup_fare_amount,
        pickup.distance_km as pickup_distance_km,
        pickup.duration_minutes as pickup_duration_minutes,
        pickup.is_current as pickup_is_current,
        dest.coord_id as dest_coord_id,
        dest.created_at as dest_created_at,
        dest.updated_at as dest_updated_at,
        dest.entity_id as dest_entity_id,
        dest.entity_type as dest_entity_type,
        dest.address as dest_address,
        dest.latitude as dest_latitude,
        dest.longitude as dest_longitude,
        dest.fare_amount as dest_fare_amount,
        dest.distance_km as dest_distance_km,
        dest.duration_minutes as dest_duration_minutes,
        dest.is_current as dest_is_current
    FROM rides r
    LEFT JOIN coordinates pickup ON r.pickup_coord_id = pickup.coord_id
    LEFT JOIN coordinates dest ON r.destination_coord_id = dest.coord_id
    WHERE r.ride_id = $1
    `

    var ride model.Rides
    var (
        // Ride timestamps (can be NULL)
        requestedAt, matchedAt, arrivedAt, startedAt, completedAt, cancelledAt *time.Time
        
        // Driver ID (can be NULL)
        driverID *string
        
        // Cancellation reason (can be NULL)
        cancellationReason *string
        
        // Coordinate timestamps
        pickupCreatedAt, pickupUpdatedAt, destCreatedAt, destUpdatedAt time.Time
    )

    err := rr.db.conn.QueryRow(ctx, q, rideID).Scan(
        &ride.ID,
        &ride.CreatedAt,
        &ride.UpdateAt,
        &ride.RideNumber,
        &ride.PassengerId,
        &driverID,
        &ride.VehicleType,
        &ride.Status,
        &ride.Priority,
        &requestedAt,
        &matchedAt,
        &arrivedAt,
        &startedAt,
        &completedAt,
        &cancelledAt,
        &cancellationReason,
        &ride.EstimatedFare,
        &ride.FinalFare,
        // Pickup coordinate
        &ride.PickupCoordinate.Id,
        &pickupCreatedAt,
        &pickupUpdatedAt,
        &ride.PickupCoordinate.EntityId,
        &ride.PickupCoordinate.EntityType,
        &ride.PickupCoordinate.Address,
        &ride.PickupCoordinate.Latitude,
        &ride.PickupCoordinate.Longitude,
        &ride.PickupCoordinate.FareAmount,
        &ride.PickupCoordinate.DistanceKm,
        &ride.PickupCoordinate.DurationMinutes,
        &ride.PickupCoordinate.IsCurrent,
        // Destination coordinate
        &ride.DestinationCoordinate.Id,
        &destCreatedAt,
        &destUpdatedAt,
        &ride.DestinationCoordinate.EntityId,
        &ride.DestinationCoordinate.EntityType,
        &ride.DestinationCoordinate.Address,
        &ride.DestinationCoordinate.Latitude,
        &ride.DestinationCoordinate.Longitude,
        &ride.DestinationCoordinate.FareAmount,
        &ride.DestinationCoordinate.DistanceKm,
        &ride.DestinationCoordinate.DurationMinutes,
        &ride.DestinationCoordinate.IsCurrent,
    )

    if err != nil {
        return model.Rides{}, err
    }

    // Handle nullable fields
    if driverID != nil {
        ride.DriverId = *driverID
    }
    
    if cancellationReason != nil {
        ride.CancellationReason = *cancellationReason
    }
    
    // Handle nullable timestamps
    if requestedAt != nil {
        ride.RequestedAt = *requestedAt
    }
    if matchedAt != nil {
        ride.MatchedAt = *matchedAt
    }
    if arrivedAt != nil {
        ride.ArrivedAt = *arrivedAt
    }
    if startedAt != nil {
        ride.StartedAt = *startedAt
    }
    if completedAt != nil {
        ride.CompletedAt = *completedAt
    }
    if cancelledAt != nil {
        ride.CancelledAt = *cancelledAt
    }

    // Set coordinate timestamps
    ride.PickupCoordinate.CreatedAt = pickupCreatedAt
    ride.PickupCoordinate.UpdatedAt = pickupUpdatedAt
    ride.DestinationCoordinate.CreatedAt = destCreatedAt
    ride.DestinationCoordinate.UpdatedAt = destUpdatedAt

    return ride, nil
}

func (rr *RidesRepo) CancelRide(ctx context.Context, rideID string, reason string, passengerID string) error {
    q := `
    UPDATE rides 
    SET status = 'CANCELLED', 
        cancellation_reason = $3,
        cancelled_at = NOW(),
        updated_at = NOW()
    WHERE ride_id = $1 
        AND passenger_id = $2
        AND status IN ('REQUESTED', 'DRIVER_ASSIGNED', 'ACCEPTED')
    `

    result, err := rr.db.conn.Exec(ctx, q, rideID, passengerID, reason)
    if err != nil {
        return err
    }

    rowsAffected := result.RowsAffected()
    if rowsAffected == 0 {
        return fmt.Errorf("ride not found or cannot be cancelled")
    }

    return nil
}