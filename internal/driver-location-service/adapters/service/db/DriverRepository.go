package db

import (
	"context"
	"ride-hail/internal/driver-location-service/core/domain/model"
	"time"
)

type DriverRepository struct {
	db *DataBase
}

func NewDriverRepository(db *DataBase) *DriverRepository {
	return &DriverRepository{db: db}
}

func (dr *DriverRepository) GoOnline(ctx context.Context, coord model.DriverCoordinates) (string, error) {
	UpdateQuery := `
      UPDATE     coordinates coord
      SET latitude = $1, longitude = $2
      FROM drivers
      WHERE coord.entity_id = drivers.driver_id AND drivers.driver_id = $3;
   `

	_, err := dr.db.GetConn().Exec(ctx, UpdateQuery, coord.Latitude, coord.Longitude, coord.Driver_id)
	if err != nil {
		return "", err
	}

	UpdateDriverStatus := `
      UPDATE drivers
      SET status = 'AVAILABLE'
      WHERE driver_id = $1;
   `

	_, err = dr.db.GetConn().Exec(ctx, UpdateDriverStatus, coord.Driver_id)
	if err != nil {
		return "", err
	}

	CreateQuery := `
      INSERT INTO driver_sessions(driver_id)
      VALUES ($1)
      RETURNING driver_session_id;
   `

	var session_id string
	err = dr.db.GetConn().QueryRow(ctx, CreateQuery, coord.Driver_id).Scan(&session_id)
	return session_id, err
}

func (dr *DriverRepository) GoOffline(ctx context.Context, driver_id string) (model.DriverOfflineResponse, error) {
	var results model.DriverOfflineResponse

	// Getting the summaries
	SelectQuery := `
      SELECT driver_session_id, extract(EPOCH from (NOW() - started_at))/3600.0, total_rides, total_earnings
      FROM driver_sessions
      WHERE driver_id = $1 AND ended_at IS NULL
      ORDER BY started_at DESC
      LIMIT 1;
   `

	err := dr.db.GetConn().QueryRow(ctx, SelectQuery, driver_id).Scan(
		&results.Session_id,
		&results.Session_summary.Duration_hours,
		&results.Session_summary.Rides_completed,
		&results.Session_summary.Earnings,
	)
	if err != nil {
		return model.DriverOfflineResponse{}, err
	}

	// Update Session ended_at time
	UpdateQuery := `
      UPDATE driver_sessions
      SET ended_at = NOW()
      WHERE driver_session_id = $1;
   `

	_, err = dr.db.GetConn().Exec(ctx, UpdateQuery, results.Session_id)
	if err != nil {
		return model.DriverOfflineResponse{}, err
	}

	// Update Driver Status
	UpdateStatusQuery := `
      UPDATE drivers
      SET status = 'OFFLINE'
      WHERE driver_id = $1;
   `

	_, err = dr.db.GetConn().Exec(ctx, UpdateStatusQuery, driver_id)
	return results, err
}

func (dr *DriverRepository) UpdateLocation(ctx context.Context, driver_id string, newLocation model.NewLocation) (model.NewLocationResponse, error) {
	// First, update the existing current coordinate or insert a new one
	// We need to handle the case where there might not be a current coordinate yet

	// Check if there's an existing current coordinate
	CheckExistingQuery := `
		SELECT coord_id FROM coordinates 
		WHERE entity_id = $1 AND entity_type = 'DRIVER' AND is_current = true
	`

	var existingCoordID string
	err := dr.db.GetConn().QueryRow(ctx, CheckExistingQuery, driver_id).Scan(&existingCoordID)

	var response model.NewLocationResponse

	if err == nil {
		// Update existing current coordinate
		UpdateQuery := `
			UPDATE coordinates 
			SET latitude = $1, longitude = $2, updated_at = NOW()
			WHERE coord_id = $3
			RETURNING coord_id, updated_at;
		`
		err = dr.db.GetConn().QueryRow(ctx, UpdateQuery, newLocation.Latitude, newLocation.Longitude, existingCoordID).Scan(&response.Coordinate_id, &response.Updated_at)
		if err != nil {
			return model.NewLocationResponse{}, err
		}
	} else {
		// Insert new current coordinate
		InsertQuery := `
			INSERT INTO coordinates (entity_id, entity_type, address, latitude, longitude, is_current)
			VALUES ($1, 'DRIVER', 'Current Location', $2, $3, true)
			RETURNING coord_id, updated_at;
		`
		err = dr.db.GetConn().QueryRow(ctx, InsertQuery, driver_id, newLocation.Latitude, newLocation.Longitude).Scan(&response.Coordinate_id, &response.Updated_at)
		if err != nil {
			return model.NewLocationResponse{}, err
		}
	}

	// Then insert into location history
	NewLocationQuery := `
      INSERT INTO location_history(coord_id, driver_id, latitude, longitude, accuracy_meters, speed_kmh, heading_degrees, ride_id)
      SELECT $1, $2, $3, $4, $5, $6, $7, 
             (SELECT ride_id FROM rides WHERE driver_id = $2 AND status IN ('EN_ROUTE', 'ARRIVED', 'IN_PROGRESS') LIMIT 1);
   `

	_, err = dr.db.GetConn().Exec(ctx, NewLocationQuery,
		response.Coordinate_id,
		driver_id,
		newLocation.Latitude,
		newLocation.Longitude,
		newLocation.Accuracy_meters,
		newLocation.Speed_kmh,
		newLocation.Heading_Degrees)
	if err != nil {
		return model.NewLocationResponse{}, err
	}

	return response, nil
}

func (dr *DriverRepository) StartRide(ctx context.Context, requestData model.StartRide) (model.StartRideResponse, error) {
	UpdateRideStatusQuery := `
      UPDATE rides
      SET status = 'IN_PROGRESS',
          started_at = NOW()
      WHERE ride_id = $1;
   `

	_, err := dr.db.GetConn().Exec(ctx, UpdateRideStatusQuery, requestData.Ride_id)
	if err != nil {
		return model.StartRideResponse{}, err
	}

	UpdateDriverStatusQuery := `
      UPDATE drivers
      SET status = 'BUSY'
      WHERE driver_id = $1;
   `

	_, err = dr.db.GetConn().Exec(ctx, UpdateDriverStatusQuery, requestData.Driver_location.Driver_id)
	if err != nil {
		return model.StartRideResponse{}, err
	}

	var response model.StartRideResponse
	response.Ride_id = requestData.Ride_id
	response.Status = "IN_PROGRESS"
	response.Started_at = time.Now().Format(time.RFC3339)
	response.Message = "Ride started successfully"
	return response, nil
}

func (dr *DriverRepository) CompleteRide(ctx context.Context, requestData model.RideCompleteForm) (model.RideCompleteResponse, error) {
	var response model.RideCompleteResponse

	// Begin transaction
	tx, err := dr.db.GetConn().Begin(ctx)
	if err != nil {
		return model.RideCompleteResponse{}, err
	}

	defer tx.Rollback(ctx)

	// Update ride status and set completion time
	RidesQuery := `
      UPDATE rides
      SET status = 'COMPLETED',
          completed_at = NOW(),
          final_fare = $2
      WHERE ride_id = $1;
   `

	// Calculate final fare (you might want to adjust this logic)
	finalFare := requestData.ActualDistancekm * 100 // Example: 100 per km
	_, err = tx.Exec(ctx, RidesQuery, requestData.Ride_id, finalFare)
	if err != nil {
		return model.RideCompleteResponse{}, err
	}

	// Update coordinates for the destination
	CoordinatesQuery := `
      UPDATE coordinates
      SET latitude = $2,
          longitude = $3
      FROM rides
      WHERE coordinates.coord_id = rides.destination_coord_id AND rides.ride_id = $1;
   `

	_, err = tx.Exec(ctx, CoordinatesQuery, requestData.Ride_id, requestData.FinalLocation.Latitude, requestData.FinalLocation.Longitude)
	if err != nil {
		return model.RideCompleteResponse{}, err
	}

	// Update driver status back to available
	UpdateDriverStatusQuery := `
      UPDATE drivers
      SET status = 'AVAILABLE',
          total_rides = total_rides + 1,
          total_earnings = total_earnings + $2
      FROM rides
      WHERE drivers.driver_id = rides.driver_id AND rides.ride_id = $1;
   `

	_, err = tx.Exec(ctx, UpdateDriverStatusQuery, requestData.Ride_id, finalFare)
	if err != nil {
		return model.RideCompleteResponse{}, err
	}

	// Update driver session earnings
	UpdateSessionQuery := `
      UPDATE driver_sessions
      SET total_rides = total_rides + 1,
          total_earnings = total_earnings + $2
      WHERE driver_id = (SELECT driver_id FROM rides WHERE ride_id = $1)
        AND ended_at IS NULL;
   `

	_, err = tx.Exec(ctx, UpdateSessionQuery, requestData.Ride_id, finalFare)
	if err != nil {
		return model.RideCompleteResponse{}, err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return model.RideCompleteResponse{}, err
	}

	response.Ride_id = requestData.Ride_id
	response.Status = "COMPLETED"
	response.CompletedAt = time.Now().Format(time.RFC3339)
	response.DriverEarning = finalFare
	response.Message = "Ride completed successfully"
	return response, nil
}
