package middleware

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"net/http"
// 	"ride-hail/internal/domain/models"
// 	"ride-hail/internal/domain/services"
// )

// type contextKey string

// const (
// 	UserContextKey contextKey = "user"
// )

// func AuthMiddleware(authService services.AuthService) func(http.Handler) http.Handler {
// 	return func(next http.Handler) http.Handler {
// 		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			authHeader := r.Header.Get("Authorization")
// 			if authHeader == "" {
// 				http.Error(w, "Authorization header required", http.StatusUnauthorized)
// 				return
// 			}

// 			if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
// 				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
// 				return
// 			}

// 			tokenString := authHeader[7:]
// 			user, err := authService.ValidateToken(tokenString)
// 			if err != nil {
// 				log.Printf("Token validation failed: %v", err)
// 				http.Error(w, "Invalid token", http.StatusUnauthorized)
// 				return
// 			}

// 			// Add user to context
// 			ctx := context.WithValue(r.Context(), UserContextKey, user)
// 			next.ServeHTTP(w, r.WithContext(ctx))
// 		})
// 	}
// }

// func RequireRole(role models.UserRole) func(http.Handler) http.Handler {
// 	return func(next http.Handler) http.Handler {
// 		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			// Get the user from context with proper type assertion
// 			ctxValue := r.Context().Value(UserContextKey)
// 			if ctxValue == nil {
// 				http.Error(w, "User not found in context", http.StatusUnauthorized)
// 				return
// 			}

// 			user, ok := ctxValue.(*models.User)
// 			if !ok {
// 				http.Error(w, "Invalid user type in context", http.StatusUnauthorized)
// 				return
// 			}

// 			if user.Role != role {
// 				http.Error(w, fmt.Sprintf("Access denied. Required role: %s", role), http.StatusForbidden)
// 				return
// 			}

// 			next.ServeHTTP(w, r)
// 		})
// 	}
// }

// func GetUserFromContext(ctx context.Context) (*models.User, bool) {
// 	ctxValue := ctx.Value(UserContextKey)
// 	if ctxValue == nil {
// 		return nil, false
// 	}

// 	user, ok := ctxValue.(*models.User)
// 	return user, ok
// }