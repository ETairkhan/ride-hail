package repo

import (
	"context"
	"ride-hail/internal/domain/models"

	"github.com/jackc/pgx/v5"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByID(ctx context.Context, userID string) (*models.User, error)
}

type userRepository struct {
	db *pgx.Conn
}

func NewUserRepository(db *pgx.Conn) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, email, role, status, password_hash)
		VALUES ($1, $2, $3, $4, $5)
	`
	
	_, err := r.db.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Role,
		user.Status,
		user.PasswordHash,
	)
	return err
}

func (r *userRepository) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	query := `
		SELECT id, created_at, updated_at, email, role, status, password_hash
		FROM users WHERE id = $1
	`
	
	var user models.User
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.Email,
		&user.Role, &user.Status, &user.PasswordHash,
	)
	
	if err != nil {
		return nil, err
	}
	return &user, nil
}