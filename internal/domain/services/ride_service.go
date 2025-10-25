package services

import (
	"context"
	"fmt"
	"ride-hail/internal/common/uuid"
	"ride-hail/internal/domain/models"
	"ride-hail/internal/domain/repo"
	"time"
)

type MessagePublisher interface {
	PublishRideRequest(ctx context.Context, ride *models.Ride, pickupCoords, destCoords *models.Coordinates) error
}

type RideService interface {
	CreateRide(ctx context.Context, req *models.CreateRideRequest) (*models.RideResponse, error)
	CancelRide(ctx context.Context, req *models.CancelRideRequest) error
	CalculateFare(vehicleType models.VehicleType, distanceKm float64, durationMin int) float64
}

type rideService struct {
	rideRepo repo.RideRepository
	userRepo repo.UserRepository
	publisher MessagePublisher
}

func NewRideService(rideRepo repo.RideRepository, userRepo repo.UserRepository, publisher MessagePublisher) RideService {
	return &rideService{
		rideRepo: rideRepo,
		userRepo: userRepo,
		publisher: publisher,
	}
}

func (s *rideService) CreateRide(ctx context.Context, req *models.CreateRideRequest) (*models.RideResponse, error) {
	// Calculate distance and duration (simplified)
	distanceKm := 5.2 // This should be calculated using proper geospatial calculation
	durationMin := 15
	
	// Calculate fare
	fare := s.CalculateFare(req.VehicleType, distanceKm, durationMin)
	
	// Create coordinates
	pickupCoords := &models.Coordinates{
		ID:         uuid.GenerateUUID(),
		EntityID:   req.PassengerID,
		EntityType: "passenger",
		Address:    req.PickupAddress,
		Latitude:   req.PickupLatitude,
		Longitude:  req.PickupLongitude,
		IsCurrent:  true,
	}
	
	destCoords := &models.Coordinates{
		ID:         uuid.GenerateUUID(),
		EntityID:   req.PassengerID,
		EntityType: "passenger",
		Address:    req.DestinationAddress,
		Latitude:   req.DestinationLatitude,
		Longitude:  req.DestinationLongitude,
		IsCurrent:  false,
	}
	
	// Save coordinates
	if err := s.rideRepo.CreateCoordinates(ctx, pickupCoords); err != nil {
		return nil, fmt.Errorf("failed to create pickup coordinates: %w", err)
	}
	
	if err := s.rideRepo.CreateCoordinates(ctx, destCoords); err != nil {
		return nil, fmt.Errorf("failed to create destination coordinates: %w", err)
	}
	
	// Create ride
	ride := &models.Ride{
		ID:                     uuid.GenerateUUID(),
		RideNumber:             generateRideNumber(),
		PassengerID:            req.PassengerID,
		VehicleType:            req.VehicleType,
		Status:                 models.StatusRequested,
		Priority:               1,
		RequestedAt:            time.Now(),
		EstimatedFare:          fare,
		PickupCoordinateID:     pickupCoords.ID,
		DestinationCoordinateID: destCoords.ID,
	}
	
	if err := s.rideRepo.CreateRide(ctx, ride); err != nil {
		return nil, fmt.Errorf("failed to create ride: %w", err)
	}
	
	// Publish to RabbitMQ
	if err := s.publisher.PublishRideRequest(ctx, ride, pickupCoords, destCoords); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to publish ride request: %v\n", err)
	}
	
	return &models.RideResponse{
		RideID:                  ride.ID,
		RideNumber:              ride.RideNumber,
		Status:                  ride.Status,
		EstimatedFare:           ride.EstimatedFare,
		EstimatedDurationMinutes: durationMin,
		EstimatedDistanceKm:     distanceKm,
	}, nil
}

func (s *rideService) CancelRide(ctx context.Context, req *models.CancelRideRequest) error {
	return s.rideRepo.CancelRide(ctx, req.RideID, req.Reason)
}

func (s *rideService) CalculateFare(vehicleType models.VehicleType, distanceKm float64, durationMin int) float64 {
	var baseFare, ratePerKm, ratePerMin float64
	
	switch vehicleType {
	case models.VehicleEconomy:
		baseFare = 500
		ratePerKm = 100
		ratePerMin = 50
	case models.VehiclePremium:
		baseFare = 800
		ratePerKm = 120
		ratePerMin = 60
	case models.VehicleXL:
		baseFare = 1000
		ratePerKm = 150
		ratePerMin = 75
	default:
		baseFare = 500
		ratePerKm = 100
		ratePerMin = 50
	}
	
	return baseFare + (distanceKm * ratePerKm) + (float64(durationMin) * ratePerMin)
}

func generateRideNumber() string {
	return fmt.Sprintf("RIDE_%s", time.Now().Format("20060102_150405"))
}