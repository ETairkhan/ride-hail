package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"ride-hail/config"
	"ride-hail/internal/common/uuid"
	"ride-hail/internal/domain/models"
	"ride-hail/internal/domain/repo"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthService interface {
	Register(ctx context.Context, req *models.RegisterRequest) (*models.AuthResponse, error)
	Login(ctx context.Context, req *models.AuthRequest) (*models.AuthResponse, error)
	ValidateToken(tokenString string) (*models.User, error)
	GetUserProfile(ctx context.Context, userID string) (*models.UserResponse, error)
}

type authService struct {
	authRepo repo.AuthRepository
	config   config.Config
}

func NewAuthService(authRepo repo.AuthRepository, config config.Config) AuthService {
	return &authService{
		authRepo: authRepo,
		config:   config,
	}
}

// hashPassword uses SHA-256 for password hashing (for demo purposes)
// In production, you'd want to use a more secure method with salt
func (s *authService) hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// verifyPassword verifies the password against the stored hash
func (s *authService) verifyPassword(password, hashedPassword string) bool {
	return s.hashPassword(password) == hashedPassword
}

func (s *authService) Register(ctx context.Context, req *models.RegisterRequest) (*models.AuthResponse, error) {
	// Check if user already exists
	existingUser, err := s.authRepo.GetUserByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, errors.New("user already exists with this email")
	}

	// Hash password
	hashedPassword := s.hashPassword(req.Password)

	// Set default role if not provided
	if req.Role == "" {
		req.Role = models.RolePassenger
	}

	// Create user
	user := &models.User{
		ID:           uuid.GenerateUUID(),
		Email:        req.Email,
		Role:         req.Role,
		Status:       models.StatusActive,
		PasswordHash: hashedPassword,
		Name:         req.Name,
		Phone:        req.Phone,
	}

	if err := s.authRepo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate JWT token
	token, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &models.AuthResponse{
		Token: token,
		User: models.UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			Role:      user.Role,
			Status:    user.Status,
			Name:      user.Name,
			Phone:     user.Phone,
			CreatedAt: time.Now(),
		},
	}, nil
}

func (s *authService) Login(ctx context.Context, req *models.AuthRequest) (*models.AuthResponse, error) {
	// Get user by email
	user, err := s.authRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Check if user is active
	if user.Status != models.StatusActive {
		return nil, errors.New("account is not active")
	}

	// Verify password
	if !s.verifyPassword(req.Password, user.PasswordHash) {
		return nil, errors.New("invalid email or password")
	}

	// Generate JWT token
	token, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &models.AuthResponse{
		Token: token,
		User: models.UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			Role:      user.Role,
			Status:    user.Status,
			Name:      user.Name,
			Phone:     user.Phone,
			CreatedAt: user.CreatedAt,
		},
	}, nil
}

func (s *authService) ValidateToken(tokenString string) (*models.User, error) {
	// Parse the token with the key function
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	// Check if token is valid and get claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, ok := claims["user_id"].(string)
		if !ok {
			return nil, errors.New("invalid token claims: user_id not found")
		}

		// Get user from repository to ensure they still exist and are active
		user, err := s.authRepo.GetUserByID(context.Background(), userID)
		if err != nil {
			return nil, errors.New("user not found")
		}

		if user.Status != models.StatusActive {
			return nil, errors.New("user account is not active")
		}

		return user, nil
	}

	return nil, errors.New("invalid token")
}

func (s *authService) GetUserProfile(ctx context.Context, userID string) (*models.UserResponse, error) {
	user, err := s.authRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return &models.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Role:      user.Role,
		Status:    user.Status,
		Name:      user.Name,
		Phone:     user.Phone,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (s *authService) generateToken(user *models.User) (string, error) {
	// Create claims with RegisteredClaims for standard fields
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    string(user.Role),
		"exp":     time.Now().Add(time.Duration(s.config.JWTExpiry) * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
		"iss":     "ride-hail-service",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}