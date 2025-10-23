package ride

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"ride-hail/internal/common/uuid"
	"ride-hail/pkg/utils"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rabbitmq/amqp091-go"
)

// User struct
type User struct {
	UserID        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Email         string
	Role          string // PASSENGER, DRIVER, ADMIN
	Status        string // ACTIVE, INACTIVE, BANNED
	PasswordHash  string
}

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

// Rides struct now includes pickup and destination coordinates
type Rides struct {
	RideID                 string
	CreatedAt              time.Time
	UpdatedAt              time.Time
	RideNumber             string
	PassengerID            string
	DriverID               string
	VehicleType            string // ECONOMY, PREMIUM, XL
	Status                 string // REQUESTED, MATCHED, EN_ROUTE, ARRIVED, IN_PROGRESS, COMPLETED, CANCELLED
	Priority               int64
	RequestedAt            time.Time
	MatchedAt              time.Time
	StartedAt              time.Time
	CompletedAt            time.Time
	CancelledAt            time.Time
	CancellationReason     string
	EstimatedFare          float64
	FinalFare              float64
	PickupCoordinates      Coordinates
	DestinationCoordinates Coordinates
}
// --------------------- MAIN SERVICE ---------------------

func StartService(config utils.Config, dbConn *pgx.Conn, rabbitConn *amqp091.Connection) {
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
		passengerId := uuid.GenerateUUID()

		rideDetails := RideDetails{
			PassengerID: passengerId,
			PickupCoordinates: Coordinates{
				Latitude:  43.238949,
				Longitude: 76.889709,
				Address:   "Pickup Location Address", // Provide the pickup address
			},
			DestinationCoordinates: Coordinates{
				Latitude:  43.222015,
				Longitude: 76.851511,
				Address:   "Destination Location Address", // Provide the destination address
			},
			RideType: "ECONOMY",
		}

		rideID, err := createRideInDB(r.Context(), dbConn, rideDetails)
		if err != nil {
			http.Error(w, fmt.Sprintf("DB error: %v", err), http.StatusInternalServerError)
			return
		}

		err = notifyRideRequestToRabbitMQ(rabbitConn, rideID, rideDetails)
		if err != nil {
			http.Error(w, fmt.Sprintf("RabbitMQ error: %v", err), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Ride created successfully with ID: %s", rideID)
	}
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

func createRideInDB(ctx context.Context, dbConn *pgx.Conn, rideDetails RideDetails) (string, error) {
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
		INSERT INTO rides (id, passenger_id, ride_number, pickup_coordinate_id, destination_coordinate_id, ride_type)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		rideID, rideDetails.PassengerID, "RIDE_001", pickupCoordID, destinationCoordID, rideDetails.RideType)
	if err != nil {
		return "", fmt.Errorf("failed to insert ride into DB: %w", err)
	}

	return rideID, nil
}

// insertCoordinates now accepts a Coordinates struct
func insertCoordinates(dbConn *pgx.Conn, entityID string, entityType string, coordinates Coordinates) (string, error) {
	// Generate a new UUID for the coordinates entry
	coordID := uuid.GenerateUUID() // Use manual UUID generation

	// Insert coordinates with the provided address
	_, err := dbConn.Exec(context.Background(), `
		INSERT INTO coordinates (id, entity_id, entity_type, latitude, longitude, address)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		coordID, entityID, entityType, coordinates.Latitude, coordinates.Longitude, coordinates.Address)
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

func notifyRideRequestToRabbitMQ(rabbitConn *amqp091.Connection, rideID string, rideDetails RideDetails) error {
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
