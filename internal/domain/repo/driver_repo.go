package repo

import (
	"context"
	"ride-hail/internal/domain/models"

	"github.com/jackc/pgx/v5"
)

type DriverRepository interface {
	CreateDriver(ctx context.Context, driver *models.Driver) error
	GetDriverByUserID(ctx context.Context, userID string) (*models.Driver, error)
	GetDriverByID(ctx context.Context, driverID string) (*models.Driver, error)
	UpdateDriver(ctx context.Context, driver *models.Driver) error
	GetDriverByLicenseNumber(ctx context.Context, licenseNumber string) (*models.Driver, error)
}

type driverRepository struct {
	db *pgx.Conn
}

func NewDriverRepository(db *pgx.Conn) DriverRepository {
	return &driverRepository{db: db}
}

func (r *driverRepository) CreateDriver(ctx context.Context, driver *models.Driver) error {
	query := `
		INSERT INTO drivers (
			id, license_number, vehicle_make, vehicle_model, vehicle_year,
			vehicle_color, license_plate, vehicle_type, status, rating,
			total_rides, total_earnings, is_verified
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.db.Exec(ctx, query,
		driver.ID,
		driver.LicenseNumber,
		driver.VehicleMake,
		driver.VehicleModel,
		driver.VehicleYear,
		driver.VehicleColor,
		driver.LicensePlate,
		driver.VehicleType,
		"OFFLINE", // Default status
		5.0,       // Default rating
		0,         // Default total rides
		0.0,       // Default total earnings
		false,     // Default is_verified
	)
	return err
}

func (r *driverRepository) GetDriverByUserID(ctx context.Context, userID string) (*models.Driver, error) {
	query := `
		SELECT d.id, d.created_at, d.updated_at, u.email,
		       d.license_number, d.vehicle_make, d.vehicle_model, d.vehicle_year,
		       d.vehicle_color, d.license_plate, d.vehicle_type, d.status,
		       d.rating, d.total_rides, d.total_earnings, d.is_verified
		FROM drivers d
		JOIN users u ON d.id = u.id
		WHERE d.id = $1
	`

	var driver models.Driver
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&driver.ID, &driver.CreatedAt, &driver.UpdatedAt, &driver.Email,
		&driver.LicenseNumber, &driver.VehicleMake, &driver.VehicleModel, &driver.VehicleYear,
		&driver.VehicleColor, &driver.LicensePlate, &driver.VehicleType, &driver.Status,
		&driver.Rating, &driver.TotalRides, &driver.TotalEarnings, &driver.IsVerified,
	)

	if err != nil {
		return nil, err
	}

	return &driver, nil
}

func (r *driverRepository) GetDriverByID(ctx context.Context, driverID string) (*models.Driver, error) {
	return r.GetDriverByUserID(ctx, driverID)
}

func (r *driverRepository) GetDriverByLicenseNumber(ctx context.Context, licenseNumber string) (*models.Driver, error) {
	query := `
		SELECT d.id, d.created_at, d.updated_at, u.email,
		       d.license_number, d.vehicle_make, d.vehicle_model, d.vehicle_year,
		       d.vehicle_color, d.license_plate, d.vehicle_type, d.status,
		       d.rating, d.total_rides, d.total_earnings, d.is_verified
		FROM drivers d
		JOIN users u ON d.id = u.id
		WHERE d.license_number = $1
	`

	var driver models.Driver
	err := r.db.QueryRow(ctx, query, licenseNumber).Scan(
		&driver.ID, &driver.CreatedAt, &driver.UpdatedAt, &driver.Email,
		&driver.LicenseNumber, &driver.VehicleMake, &driver.VehicleModel, &driver.VehicleYear,
		&driver.VehicleColor, &driver.LicensePlate, &driver.VehicleType, &driver.Status,
		&driver.Rating, &driver.TotalRides, &driver.TotalEarnings, &driver.IsVerified,
	)

	if err != nil {
		return nil, err
	}

	return &driver, nil
}

func (r *driverRepository) UpdateDriver(ctx context.Context, driver *models.Driver) error {
	query := `
		UPDATE drivers 
		SET license_number = $1, vehicle_make = $2, vehicle_model = $3, vehicle_year = $4,
		    vehicle_color = $5, license_plate = $6, vehicle_type = $7, status = $8,
		    rating = $9, total_rides = $10, total_earnings = $11, is_verified = $12,
		    updated_at = NOW()
		WHERE id = $13
	`

	_, err := r.db.Exec(ctx, query,
		driver.LicenseNumber,
		driver.VehicleMake,
		driver.VehicleModel,
		driver.VehicleYear,
		driver.VehicleColor,
		driver.LicensePlate,
		driver.VehicleType,
		driver.Status,
		driver.Rating,
		driver.TotalRides,
		driver.TotalEarnings,
		driver.IsVerified,
		driver.ID,
	)
	return err
}
