package models

import "time"

// User struct
type User struct {
	UserID       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Email        string
	Role         string // PASSENGER, DRIVER, ADMIN
	Status       string // ACTIVE, INACTIVE, BANNED
	PasswordHash string
}
