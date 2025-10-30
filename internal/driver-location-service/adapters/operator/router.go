package operator

import (
	"net/http"

	"ride-hail/internal/driver-location-service/adapters/operator/handlers"
	"ride-hail/internal/driver-location-service/adapters/operator/middleware"

	"ride-hail/internal/config"
)

func Router(handlers *handlers.Handlers, cfg *config.Config) http.Handler {
	mux := http.NewServeMux()
	mdl := middleware.NewAuthMiddleware(cfg.App.PublicJwtSecret)
	mux.HandleFunc("/ws/drivers/{driver_id}", handlers.WebSocketHandler.HandleDriverWebSocket)
	mux.Handle("/drivers/{driver_id}/online", mdl.SessionHandler(func() http.HandlerFunc { return handlers.DriverHandler.GoOnline }()))
	mux.Handle("/drivers/{driver_id}/offline", mdl.SessionHandler(func() http.HandlerFunc { return handlers.DriverHandler.GoOffline }()))
	mux.Handle("/drivers/{driver_id}/location", mdl.SessionHandler(func() http.HandlerFunc { return handlers.DriverHandler.UpdateLocation }()))
	mux.Handle("/drivers/{driver_id}/start", mdl.SessionHandler(func() http.HandlerFunc { return handlers.DriverHandler.StartRide }()))
	mux.Handle("/drivers/{driver_id}/complete", mdl.SessionHandler(func() http.HandlerFunc { return handlers.DriverHandler.CompleteRide }()))

	return mux
}
