package repo

import (
	"context"
	"ride-hail/internal/domain/models"

	"github.com/jackc/pgx/v5"
)

type AuthRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, userID string) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
}

type authRepository struct {
	db *pgx.Conn
}

func NewAuthRepository(db *pgx.Conn) AuthRepository {
	return &authRepository{db: db}
}

func (r *authRepository) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, email, role, status, password_hash, attrs)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	
	// Store additional fields in attrs JSONB
	attrs := map[string]interface{}{
		"name":  user.Name,
		"phone": user.Phone,
	}
	
	_, err := r.db.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Role,
		user.Status,
		user.PasswordHash,
		attrs,
	)
	return err
}

func (r *authRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, created_at, updated_at, email, role, status, password_hash, attrs
		FROM users WHERE email = $1
	`
	
	var user models.User
	var attrs map[string]interface{}
	
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.Email,
		&user.Role, &user.Status, &user.PasswordHash, &attrs,
	)
	
	if err != nil {
		return nil, err
	}
	
	// Extract additional fields from attrs
	if attrs != nil {
		if name, ok := attrs["name"].(string); ok {
			user.Name = name
		}
		if phone, ok := attrs["phone"].(string); ok {
			user.Phone = phone
		}
	}
	
	return &user, nil
}

func (r *authRepository) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	query := `
		SELECT id, created_at, updated_at, email, role, status, password_hash, attrs
		FROM users WHERE id = $1
	`
	
	var user models.User
	var attrs map[string]interface{}
	
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.Email,
		&user.Role, &user.Status, &user.PasswordHash, &attrs,
	)
	
	if err != nil {
		return nil, err
	}
	
	// Extract additional fields from attrs
	if attrs != nil {
		if name, ok := attrs["name"].(string); ok {
			user.Name = name
		}
		if phone, ok := attrs["phone"].(string); ok {
			user.Phone = phone
		}
	}
	
	return &user, nil
}

func (r *authRepository) UpdateUser(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users 
		SET email = $1, role = $2, status = $3, password_hash = $4, attrs = $5, updated_at = NOW()
		WHERE id = $6
	`
	
	attrs := map[string]interface{}{
		"name":  user.Name,
		"phone": user.Phone,
	}
	
	_, err := r.db.Exec(ctx, query,
		user.Email,
		user.Role,
		user.Status,
		user.PasswordHash,
		attrs,
		user.ID,
	)
	return err
}