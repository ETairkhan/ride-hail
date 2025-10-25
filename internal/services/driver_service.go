package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ride-hail/internal/common/config"
	"ride-hail/internal/database"
	"ride-hail/models"
	"ride-hail/websocket"

	"github.com/jackc/pgx/v5"
	"github.com/rabbitmq/amqp091-go"
)

type DriverService struct {
	db       *database.DB
	rabbitMQ *database.RabbitMQ
	wsHub    *websocket.Hub
	cfg      *config.Config
}

func NewDriverService(db *database.DB, rabbitMQ *database.RabbitMQ, wsHub *websocket.Hub, cfg *config.Config) *DriverService {
	ds := &DriverService{
		db:       db,
		rabbitMQ: rabbitMQ,
		wsHub:    wsHub,
		cfg:      cfg,
	}

	// Start consuming messages only if RabbitMQ is available
	if rabbitMQ != nil {
		go ds.consumeRideRequests()
		go ds.consumeRideStatusUpdates()
	} else {
		log.Println("RabbitMQ not available - message consuming disabled")
	}

	return ds
}

type OnlineResponse struct {
	Status    string `json:"status"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type SessionSummary struct {
	DurationHours  float64 `json:"duration_hours"`
	RidesCompleted int     `json:"rides_completed"`
	Earnings       float64 `json:"earnings"`
}

type OfflineResponse struct {
	Status         string          `json:"status"`
	SessionID      string          `json:"session_id"`
	SessionSummary *SessionSummary `json:"session_summary"`
	Message        string          `json:"message"`
}

func (ds *DriverService) GoOnline(ctx context.Context, driverID string, lat, lng float64) (*OnlineResponse, error) {
	tx, err := ds.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Update driver status to AVAILABLE
	result, err := tx.Exec(ctx, `
		UPDATE drivers 
		SET status = 'AVAILABLE', updated_at = $1 
		WHERE id = $2 AND status != 'BANNED'
	`, time.Now(), driverID)
	if err != nil {
		return nil, fmt.Errorf("failed to update driver status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return nil, fmt.Errorf("driver not found or banned")
	}

	// Create new driver session
	var sessionID string
	err = tx.QueryRow(ctx, `
		INSERT INTO driver_sessions (driver_id, started_at) 
		VALUES ($1, $2) 
		RETURNING id
	`, driverID, time.Now()).Scan(&sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create driver session: %w", err)
	}

	// Update driver's current location
	_, err = tx.Exec(ctx, `
		UPDATE coordinates 
		SET is_current = false 
		WHERE entity_id = $1 AND entity_type = 'driver' AND is_current = true
	`, driverID)
	if err != nil {
		return nil, fmt.Errorf("failed to update previous coordinates: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO coordinates (entity_id, entity_type, address, latitude, longitude, is_current)
		VALUES ($1, 'driver', 'Online location', $2, $3, true)
	`, driverID, lat, lng)
	if err != nil {
		return nil, fmt.Errorf("failed to insert new coordinates: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Publish driver status update
	ds.publishDriverStatus(driverID, "AVAILABLE", "")

	return &OnlineResponse{
		Status:    "AVAILABLE",
		SessionID: sessionID,
		Message:   "You are now online and ready to accept rides",
	}, nil
}

func (ds *DriverService) GoOffline(ctx context.Context, driverID string) (*OfflineResponse, error) {
	tx, err := ds.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Update driver status to OFFLINE
	result, err := tx.Exec(ctx, `
		UPDATE drivers 
		SET status = 'OFFLINE', updated_at = $1 
		WHERE id = $2
	`, time.Now(), driverID)
	if err != nil {
		return nil, fmt.Errorf("failed to update driver status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return nil, fmt.Errorf("driver not found")
	}

	// End the current driver session
	var sessionSummary SessionSummary
	var sessionID string
	err = tx.QueryRow(ctx, `
		UPDATE driver_sessions 
		SET ended_at = $1,
		    total_rides = (
		        SELECT COUNT(*) FROM rides 
		        WHERE driver_id = $2 AND completed_at BETWEEN started_at AND $1
		    ),
		    total_earnings = (
		        SELECT COALESCE(SUM(final_fare), 0) FROM rides 
		        WHERE driver_id = $2 AND completed_at BETWEEN started_at AND $1
		    )
		WHERE driver_id = $2 AND ended_at IS NULL
		RETURNING id, 
		          EXTRACT(EPOCH FROM ($1 - started_at)) / 3600 as duration_hours,
		          total_rides,
		          total_earnings
	`, time.Now(), driverID).Scan(&sessionID, &sessionSummary.DurationHours, &sessionSummary.RidesCompleted, &sessionSummary.Earnings)
	if err != nil {
		// If no active session, create a summary with zeros
		if err == pgx.ErrNoRows {
			sessionSummary = SessionSummary{
				DurationHours:  0,
				RidesCompleted: 0,
				Earnings:       0,
			}
			// Create a session ID for the response
			sessionID = "no_active_session"
		} else {
			return nil, fmt.Errorf("failed to update driver session: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Publish driver status update
	ds.publishDriverStatus(driverID, "OFFLINE", "")

	return &OfflineResponse{
		Status:         "OFFLINE",
		SessionID:      sessionID,
		SessionSummary: &sessionSummary,
		Message:        "You are now offline",
	}, nil
}

func (ds *DriverService) UpdateLocation(ctx context.Context, driverID string, update models.LocationUpdate) (*models.Coordinate, error) {
	tx, err := ds.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Get current ride if any
	var currentRideID *string
	err = tx.QueryRow(ctx, `
		SELECT id FROM rides 
		WHERE driver_id = $1 AND status IN ('MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')
		LIMIT 1
	`, driverID).Scan(&currentRideID)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to get current ride: %w", err)
	}

	// Update previous coordinates to not current
	_, err = tx.Exec(ctx, `
		UPDATE coordinates 
		SET is_current = false 
		WHERE entity_id = $1 AND entity_type = 'driver' AND is_current = true
	`, driverID)
	if err != nil {
		return nil, fmt.Errorf("failed to update previous coordinates: %w", err)
	}

	// Insert new current coordinate
	var coord models.Coordinate
	err = tx.QueryRow(ctx, `
		INSERT INTO coordinates (entity_id, entity_type, address, latitude, longitude, is_current)
		VALUES ($1, 'driver', $2, $3, $4, true)
		RETURNING id, created_at, updated_at, entity_id, entity_type, address, latitude, longitude, is_current
	`, driverID, update.Address, update.Latitude, update.Longitude).Scan(
		&coord.ID, &coord.CreatedAt, &coord.UpdatedAt, &coord.EntityID, &coord.EntityType,
		&coord.Address, &coord.Latitude, &coord.Longitude, &coord.IsCurrent,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert new coordinate: %w", err)
	}

	// Insert into location history
	_, err = tx.Exec(ctx, `
		INSERT INTO location_history (coordinate_id, driver_id, latitude, longitude, accuracy_meters, speed_kmh, heading_degrees, ride_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, coord.ID, driverID, update.Latitude, update.Longitude, update.AccuracyMeters, update.SpeedKMH, update.HeadingDegrees, currentRideID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert location history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Broadcast location update
	locationMsg := models.LocationMessage{
		DriverID: driverID,
		RideID:   getStringValue(currentRideID),
		Location: models.Location{
			Latitude:  update.Latitude,
			Longitude: update.Longitude,
		},
		SpeedKMH:       update.SpeedKMH,
		HeadingDegrees: update.HeadingDegrees,
		Timestamp:      time.Now(),
	}

	ds.broadcastLocationUpdate(locationMsg)

	return &coord, nil
}

func (ds *DriverService) StartRide(ctx context.Context, driverID, rideID string, location models.Location) (map[string]interface{}, error) {
	tx, err := ds.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Update ride status to IN_PROGRESS
	result, err := tx.Exec(ctx, `
		UPDATE rides 
		SET status = 'IN_PROGRESS', started_at = $1, updated_at = $1
		WHERE id = $2 AND driver_id = $3
	`, time.Now(), rideID, driverID)
	if err != nil {
		return nil, fmt.Errorf("failed to update ride status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return nil, fmt.Errorf("ride not found or driver mismatch")
	}

	// Update driver status to BUSY
	_, err = tx.Exec(ctx, `
		UPDATE drivers 
		SET status = 'BUSY', updated_at = $1 
		WHERE id = $2
	`, time.Now(), driverID)
	if err != nil {
		return nil, fmt.Errorf("failed to update driver status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Publish status updates
	ds.publishRideStatus(rideID, "IN_PROGRESS", driverID)
	ds.publishDriverStatus(driverID, "BUSY", rideID)

	return map[string]interface{}{
		"ride_id":    rideID,
		"status":     "BUSY",
		"started_at": time.Now(),
		"message":    "Ride started successfully",
	}, nil
}

func (ds *DriverService) CompleteRide(ctx context.Context, driverID, rideID string, location models.Location, actualDistanceKM float64, actualDurationMinutes int) (map[string]interface{}, error) {
	tx, err := ds.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Calculate final fare (simplified - in real implementation, use pricing logic)
	var estimatedFare float64
	err = tx.QueryRow(ctx, `
		SELECT estimated_fare FROM rides WHERE id = $1
	`, rideID).Scan(&estimatedFare)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride details: %w", err)
	}

	finalFare := estimatedFare         // In real implementation, adjust based on actual distance/duration
	driverEarnings := finalFare * 0.75 // 75% to driver, 25% to platform

	// Update ride status to COMPLETED
	result, err := tx.Exec(ctx, `
		UPDATE rides 
		SET status = 'COMPLETED', completed_at = $1, final_fare = $2, updated_at = $1
		WHERE id = $3 AND driver_id = $4
	`, time.Now(), finalFare, rideID, driverID)
	if err != nil {
		return nil, fmt.Errorf("failed to update ride status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return nil, fmt.Errorf("ride not found or driver mismatch")
	}

	// Update driver status back to AVAILABLE and increment stats
	_, err = tx.Exec(ctx, `
		UPDATE drivers 
		SET status = 'AVAILABLE', 
		    total_rides = total_rides + 1,
		    total_earnings = total_earnings + $1,
		    updated_at = $2
		WHERE id = $3
	`, driverEarnings, time.Now(), driverID)
	if err != nil {
		return nil, fmt.Errorf("failed to update driver status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Publish status updates
	ds.publishRideStatus(rideID, "COMPLETED", driverID)
	ds.publishDriverStatus(driverID, "AVAILABLE", "")

	return map[string]interface{}{
		"ride_id":         rideID,
		"status":          "AVAILABLE",
		"completed_at":    time.Now(),
		"driver_earnings": driverEarnings,
		"message":         "Ride completed successfully",
	}, nil
}

func (ds *DriverService) FindNearbyDrivers(ctx context.Context, pickupLat, pickupLng float64, vehicleType string, maxDistanceKM float64) ([]models.Driver, error) {
	query := `
		SELECT d.id, u.email, d.rating, c.latitude, c.longitude,
		       ST_Distance(
		         ST_MakePoint(c.longitude, c.latitude)::geography,
		         ST_MakePoint($1, $2)::geography
		       ) / 1000 as distance_km
		FROM drivers d
		JOIN users u ON d.id = u.id
		JOIN coordinates c ON c.entity_id = d.id
		  AND c.entity_type = 'driver'
		  AND c.is_current = true
		WHERE d.status = 'AVAILABLE'
		  AND d.vehicle_type = $3
		  AND ST_DWithin(
		        ST_MakePoint(c.longitude, c.latitude)::geography,
		        ST_MakePoint($1, $2)::geography,
		        $4 * 1000  -- Convert km to meters
		      )
		ORDER BY distance_km, d.rating DESC
		LIMIT 10
	`

	rows, err := ds.db.Pool.Query(ctx, query, pickupLng, pickupLat, vehicleType, maxDistanceKM)
	if err != nil {
		return nil, fmt.Errorf("failed to query nearby drivers: %w", err)
	}
	defer rows.Close()

	var drivers []models.Driver
	for rows.Next() {
		var driver models.Driver
		err := rows.Scan(&driver.ID, &driver.Email, &driver.Rating, &driver.Latitude, &driver.Longitude, &driver.DistanceKM)
		if err != nil {
			return nil, fmt.Errorf("failed to scan driver: %w", err)
		}
		drivers = append(drivers, driver)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return drivers, nil
}

func (ds *DriverService) consumeRideRequests() {
	if ds.rabbitMQ == nil {
		log.Println("RabbitMQ not available, cannot consume ride requests")
		return
	}

	ch, err := ds.rabbitMQ.Conn.Channel()
	if err != nil {
		log.Printf("Failed to open RabbitMQ channel: %v", err)
		return
	}
	defer ch.Close()

	// Ensure exchange exists
	err = ch.ExchangeDeclare(
		"ride_topic", // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		log.Printf("Failed to declare exchange: %v", err)
		return
	}

	// Declare queue for driver matching
	q, err := ch.QueueDeclare(
		"driver_matching", // name
		true,              // durable
		false,             // delete when unused
		false,             // exclusive
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		log.Printf("Failed to declare queue: %v", err)
		return
	}

	// Bind queue to exchange for ride requests
	err = ch.QueueBind(
		q.Name,           // queue name
		"ride.request.*", // routing key
		"ride_topic",     // exchange
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		log.Printf("Failed to bind queue: %v", err)
		return
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Printf("Failed to register consumer: %v", err)
		return
	}

	log.Println("Started consuming ride requests from RabbitMQ")

	for msg := range msgs {
		var rideRequest models.RideRequest
		if err := json.Unmarshal(msg.Body, &rideRequest); err != nil {
			log.Printf("Failed to unmarshal ride request: %v", err)
			continue
		}

		ctx := context.Background()
		ds.handleRideRequest(ctx, rideRequest)
	}
}

func (ds *DriverService) handleRideRequest(ctx context.Context, request models.RideRequest) {
	// Find nearby drivers - use VehicleType instead of RideType
	drivers, err := ds.FindNearbyDrivers(ctx,
		request.PickupLocation.Latitude,
		request.PickupLocation.Longitude,
		request.VehicleType, // Fixed: Use VehicleType instead of RideType
		request.MaxDistanceKM,
	)
	if err != nil {
		log.Printf("Failed to find nearby drivers: %v", err)
		return
	}

	// Send ride offers to drivers via WebSocket
	for _, driver := range drivers {
		offer := models.RideOffer{
			OfferID:    fmt.Sprintf("offer_%s_%s", request.RideID, driver.ID),
			RideID:     request.RideID,
			RideNumber: request.RideNumber,
			PickupLocation: models.LocationWithAddress{
				Location: request.PickupLocation,
				Address:  request.PickupAddress,
			},
			DestinationLocation: models.LocationWithAddress{
				Location: request.DestinationLocation,
				Address:  request.DestinationAddress,
			},
			EstimatedFare:                request.EstimatedFare,
			DriverEarnings:               request.EstimatedFare * 0.75, // 75% to driver
			DistanceToPickupKM:           driver.DistanceKM,
			EstimatedRideDurationMinutes: request.EstimatedDurationMinutes,
			ExpiresAt:                    time.Now().Add(time.Duration(request.TimeoutSeconds) * time.Second),
		}

		// Send offer via WebSocket
		ds.wsHub.SendToDriver(driver.ID, "ride_offer", offer)
	}
}

func (ds *DriverService) consumeRideStatusUpdates() {
	if ds.rabbitMQ == nil {
		log.Println("RabbitMQ not available, cannot consume ride status updates")
		return
	}

	// Similar implementation to consumeRideRequests but for ride status updates
	log.Println("Starting ride status updates consumer...")
	// Implementation would go here
}

func (ds *DriverService) publishDriverStatus(driverID, status, rideID string) {
	if ds.rabbitMQ == nil {
		return // Skip if RabbitMQ is not available
	}

	// Publish driver status update to RabbitMQ
	statusMsg := models.DriverStatusMessage{
		DriverID:  driverID,
		Status:    status,
		RideID:    rideID,
		Timestamp: time.Now(),
	}

	ds.publishToRabbitMQ("driver_topic", fmt.Sprintf("driver.status.%s", driverID), statusMsg)
}

func (ds *DriverService) publishRideStatus(rideID, status, driverID string) {
	if ds.rabbitMQ == nil {
		return // Skip if RabbitMQ is not available
	}

	// Publish ride status update to RabbitMQ
	statusMsg := models.RideStatusMessage{
		RideID:    rideID,
		Status:    status,
		DriverID:  driverID,
		Timestamp: time.Now(),
	}

	ds.publishToRabbitMQ("ride_topic", fmt.Sprintf("ride.status.%s", status), statusMsg)
}

func (ds *DriverService) broadcastLocationUpdate(location models.LocationMessage) {
	if ds.rabbitMQ == nil {
		return // Skip if RabbitMQ is not available
	}

	// Broadcast location update via fanout exchange
	ds.publishToRabbitMQ("location_fanout", "", location)
}

func (ds *DriverService) publishToRabbitMQ(exchange, routingKey string, message interface{}) {
	if ds.rabbitMQ == nil {
		return // Skip if RabbitMQ is not available
	}

	ch, err := ds.rabbitMQ.Conn.Channel()
	if err != nil {
		log.Printf("Failed to open channel for publishing: %v", err)
		return
	}
	defer ch.Close()

	body, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	err = ch.Publish(
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
			Timestamp:   time.Now(),
		})
	if err != nil {
		log.Printf("Failed to publish message: %v", err)
	}
}

// Helper function to safely get string value from pointer
func getStringValue(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}
