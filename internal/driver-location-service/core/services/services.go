package services

import (
	"ride-hail/internal/driver-location-service/adapters/service/database"
	ports "ride-hail/internal/driver-location-service/core/ports/driven"
	"ride-hail/internal/logger"
)

type Service struct {
	DriverService *DriverService
	AuthService   *AuthService
}

// Must properly implement Auth Service
func New(repositories *database.Repository, log logger.Logger, broker ports.IDriverBroker, secretKey string) *Service {
	return &Service{
		DriverService: NewDriverService(repositories.DriverRepository, log, broker),
		AuthService:   NewAuthService(secretKey),
	}
}
