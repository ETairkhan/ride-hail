package models

import "time"

type UserRole string
type UserStatus string

const (
	RolePassenger UserRole = "PASSENGER"
	RoleDriver    UserRole = "DRIVER"
	RoleAdmin     UserRole = "ADMIN"
	
	StatusActive   UserStatus = "ACTIVE"
	StatusInactive UserStatus = "INACTIVE"
	StatusBanned   UserStatus = "BANNED"
)

type User struct {
	ID           string     `json:"id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Email        string     `json:"email"`
	Role         UserRole   `json:"role"`
	Status       UserStatus `json:"status"`
	PasswordHash string     `json:"-"`
	Name         string     `json:"name,omitempty"`
	Phone        string     `json:"phone,omitempty"`
}

type UserResponse struct {
	ID        string     `json:"id"`
	Email     string     `json:"email"`
	Role      UserRole   `json:"role"`
	Status    UserStatus `json:"status"`
	Name      string     `json:"name,omitempty"`
	Phone     string     `json:"phone,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}