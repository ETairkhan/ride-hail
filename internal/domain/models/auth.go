package models

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type RegisterRequest struct {
	Email    string   `json:"email"`
	Password string   `json:"password"`
	Role     UserRole `json:"role,omitempty"`
	Name     string   `json:"name,omitempty"`
	Phone    string   `json:"phone,omitempty"`
}

// DriverRegisterRequest includes driver-specific fields
type DriverRegisterRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	Name          string `json:"name,omitempty"`
	Phone         string `json:"phone,omitempty"`
	LicenseNumber string `json:"license_number"`
	VehicleMake   string `json:"vehicle_make"`
	VehicleModel  string `json:"vehicle_model"`
	VehicleYear   int    `json:"vehicle_year"`
	VehicleColor  string `json:"vehicle_color"`
	LicensePlate  string `json:"license_plate"`
	VehicleType   string `json:"vehicle_type"` // e.g., "SEDAN", "SUV", "LUXURY"
}

type TokenClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

// User models remain in user.go but we'll add some fields
