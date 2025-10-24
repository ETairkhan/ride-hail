package services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"ride-hail/config"
	"ride-hail/internal/common/uuid"
	"ride-hail/internal/domain/models"

	"github.com/jackc/pgx/v5"
	"github.com/rabbitmq/amqp091-go"
)

// --------------------- MAIN SERVICE ---------------------

func StartService(config config.Config, dbConn *pgx.Conn, rabbitConn *amqp091.Connection) {
	http.HandleFunc("/rides", createRideHandler(dbConn, rabbitConn))
	http.HandleFunc("/rides/{ride_id}/cancel", cancelRideHandler(dbConn, rabbitConn))

	log.Println("Starting Ride Service on port", config.RideServicePort)
	err := http.ListenAndServe(fmt.Sprintf(":%s", config.RideServicePort), nil)
	if err != nil {
		log.Fatalf("Error starting Ride Service: %v", err)
	}
}

// --------------------- CREATE RIDE ---------------------
func createRideHandler(dbConn *pgx.Conn, rabbitConn *amqp091.Connection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create a new user ID (or fetch from the database)
		passengerId := uuid.GenerateUUID()

		// Insert the new user into the database
		err := insertUser(dbConn, passengerId)
		if err != nil {
			http.Error(w, fmt.Sprintf("DB error: %v", err), http.StatusInternalServerError)
			return
		}

		// Create RideDetails
		rideDetails := models.Rides{
			PassengerID: passengerId,
			PickupCoordinates: models.Coordinates{
				Latitude:  43.238949,
				Longitude: 76.889709,
				Address:   "Pickup Location Address",
			},
			DestinationCoordinates: models.Coordinates{
				Latitude:  43.222015,
				Longitude: 76.851511,
				Address:   "Destination Location Address",
			},
			VehicleType: "ECONOMY",
		}

		// Create ride in database
		rideID, err := createRideInDB(r.Context(), dbConn, rideDetails)
		if err != nil {
			http.Error(w, fmt.Sprintf("DB error: %v", err), http.StatusInternalServerError)
			return
		}

		// Notify RabbitMQ
		err = notifyRideRequestToRabbitMQ(rabbitConn, rideID, rideDetails)
		if err != nil {
			http.Error(w, fmt.Sprintf("RabbitMQ error: %v", err), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Ride created successfully with ID: %s", rideID)
	}
}

// Insert a new user into the users table (if needed)
func insertUser(dbConn *pgx.Conn, userID string) error {
	// Insert a new user into the users table (add other necessary fields like email, role, etc.)
	_, err := dbConn.Exec(context.Background(), `
		INSERT INTO users (id, created_at, updated_at, email, role, status, password_hash)
		VALUES ($1, now(), now(), $2, $3, 'ACTIVE', $4)`,
		userID, "user@example.com", "PASSENGER", "hashed_password")
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

// --------------------- CANCEL RIDE ---------------------

func cancelRideHandler(dbConn *pgx.Conn, rabbitConn *amqp091.Connection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rideID := "some-ride-id"

		err := cancelRideInDB(r.Context(), dbConn, rideID)
		if err != nil {
			http.Error(w, fmt.Sprintf("DB error: %v", err), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Ride %s canceled successfully", rideID)
	}
}

// --------------------- DATABASE ---------------------

func createRideInDB(ctx context.Context, dbConn *pgx.Conn, rideDetails models.Rides) (string, error) {
	// Insert pickup coordinates into coordinates table
	pickupCoordID, err := insertCoordinates(dbConn, rideDetails.PassengerID, "passenger", rideDetails.PickupCoordinates)
	if err != nil {
		return "", fmt.Errorf("failed to insert pickup coordinates: %w", err)
	}

	// Insert destination coordinates into coordinates table
	destinationCoordID, err := insertCoordinates(dbConn, rideDetails.PassengerID, "passenger", rideDetails.DestinationCoordinates)
	if err != nil {
		return "", fmt.Errorf("failed to insert destination coordinates: %w", err)
	}

	// Insert ride data into rides table, using the coordinate IDs
	rideID := uuid.GenerateUUID() // Use manual UUID generation
	_, err = dbConn.Exec(ctx, `
		INSERT INTO rides (id, passenger_id, ride_number, pickup_coordinate_id, destination_coordinate_id, vehicle_type)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		rideID, rideDetails.PassengerID, "RIDE_001", pickupCoordID, destinationCoordID, rideDetails.VehicleType)
	if err != nil {
		return "", fmt.Errorf("failed to insert ride into DB: %w", err)
	}

	return rideID, nil
}

// Insert coordinates into the coordinates table
func insertCoordinates(dbConn *pgx.Conn, entityID string, entityType string, coordinates models.Coordinates) (string, error) {
	// Generate a new UUID for the coordinates entry
	coordID := uuid.GenerateUUID() // Use manual UUID generation

	// Insert coordinates into the database
	_, err := dbConn.Exec(context.Background(), `
		INSERT INTO coordinates (id, entity_id, entity_type, latitude, longitude, address, fare_amount, distance_km, duration_minutes, is_current)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		coordID, entityID, entityType, coordinates.Latitude, coordinates.Longitude, coordinates.Address,
		coordinates.FareAmount, coordinates.DistanceKm, coordinates.DurationMinutes, coordinates.IsCurrent)
	if err != nil {
		return "", fmt.Errorf("failed to insert coordinates: %w", err)
	}

	return coordID, nil
}

func cancelRideInDB(ctx context.Context, dbConn *pgx.Conn, rideID string) error {
	_, err := dbConn.Exec(ctx, `UPDATE rides SET status = 'CANCELLED' WHERE id = $1`, rideID)
	if err != nil {
		return fmt.Errorf("failed to cancel ride in DB: %w", err)
	}
	return nil
}

// --------------------- RABBITMQ ---------------------

func notifyRideRequestToRabbitMQ(rabbitConn *amqp091.Connection, rideID string, rideDetails models.Rides) error {
	ch, err := rabbitConn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %w", err)
	}
	defer ch.Close()

	message := fmt.Sprintf(`{"ride_id":"%s","pickup_latitude":%f,"pickup_longitude":%f}`,
		rideID, rideDetails.PickupCoordinates.Latitude, rideDetails.PickupCoordinates.Longitude)

	err = ch.Publish(
		"ride_topic",   // exchange
		"ride.request", // routing key
		false,
		false,
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        []byte(message),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to send message to RabbitMQ: %w", err)
	}

	return nil
}
