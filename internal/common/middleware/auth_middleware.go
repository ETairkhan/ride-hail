package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	UserIDContextKey contextKey = "user_id"
	RoleContextKey   contextKey = "role"
)

type AuthMiddleware struct {
	accessSecret string
}

func NewAuthMiddleware(accessSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		accessSecret: accessSecret,
	}
}

func (am *AuthMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Empty JWT-Token", http.StatusBadRequest)
			return
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return []byte(am.accessSecret), nil
		})

		if err != nil {
			http.Error(w, "Failed to parse JWT-Token", http.StatusBadRequest)
			return
		}

		if !token.Valid {
			http.Error(w, "Invalid JWT-Token", http.StatusBadRequest)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Invalid claims", http.StatusUnauthorized)
			return
		}

		userID, ok := claims["user_id"].(string)
		if !ok {
			http.Error(w, "Username not found in token", http.StatusUnauthorized)
			return
		}

		role, ok := claims["role"].(string)
		if !ok {
			http.Error(w, "role not found in token", http.StatusUnauthorized)
			return
		}

		// Add user info to context instead of headers
		ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
		ctx = context.WithValue(ctx, RoleContextKey, role)
		
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Helper function to get user from context
func GetUserFromContext(ctx context.Context) (string, string, bool) {
	// Get user ID from context
	userIDValue := ctx.Value(UserIDContextKey)
	if userIDValue == nil {
		return "", "", false
	}
	
	userID, ok1 := userIDValue.(string)
	if !ok1 {
		return "", "", false
	}

	// Get role from context
	roleValue := ctx.Value(RoleContextKey)
	if roleValue == nil {
		return "", "", false
	}
	
	role, ok2 := roleValue.(string)
	if !ok2 {
		return "", "", false
	}

	return userID, role, true
}