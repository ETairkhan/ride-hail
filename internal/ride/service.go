package ride

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"ride-hail/pkg/utils"

	"github.com/jackc/pgx/v5"
	"github.com/rabbitmq/amqp091-go"
)

// Определяем структуру для данных о поездке
type RideDetails struct {
	PassengerID          string
	PickupLatitude       float64
	PickupLongitude      float64
	DestinationLatitude  float64
	DestinationLongitude float64
	RideType             string
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
		rideDetails := RideDetails{
			PassengerID:          "user-001",
			PickupLatitude:       43.238949,
			PickupLongitude:      76.889709,
			DestinationLatitude:  43.222015,
			DestinationLongitude: 76.851511,
			RideType:             "ECONOMY",
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
	rideID := "ride_12345" // Для теста, потом можно заменить на uuid_generate_v4()

	_, err := dbConn.Exec(ctx, `
		INSERT INTO rides (id, passenger_id, pickup_latitude, pickup_longitude, destination_latitude, destination_longitude, ride_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		rideID, rideDetails.PassengerID, rideDetails.PickupLatitude, rideDetails.PickupLongitude,
		rideDetails.DestinationLatitude, rideDetails.DestinationLongitude, rideDetails.RideType)
	if err != nil {
		return "", fmt.Errorf("failed to insert ride into DB: %w", err)
	}
	return rideID, nil
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
		rideID, rideDetails.PickupLatitude, rideDetails.PickupLongitude)

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
