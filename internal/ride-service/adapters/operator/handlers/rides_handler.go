// handlers/rides_handler.go
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"ride-hail/internal/mylogger"
	"ride-hail/internal/ride-service/core/domain/dto"
	"ride-hail/internal/ride-service/core/ports"
	"strings"
)

type RidesHandler struct {
	ridesService ports.IRidesService
	log          mylogger.Logger
}

func NewRidesHandler(rs ports.IRidesService, log mylogger.Logger) *RidesHandler {
	return &RidesHandler{
		ridesService: rs,
		log:          log,
	}
}

func (rh *RidesHandler) CreateRide() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := dto.RidesRequestDto{}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res, err := rh.ridesService.CreateRide(req)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}

		jsonResponse(w, http.StatusAccepted, res)
	}
}

func (rh *RidesHandler) CancelRide() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract ride_id from URL path
		rideID := r.PathValue("ride_id")
		if rideID == "" {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("ride_id is required"))
			return
		}

		// Extract passenger_id from headers (set by auth middleware)
		passengerID := r.Header.Get("X-UserId")
		if passengerID == "" {
			jsonError(w, http.StatusUnauthorized, fmt.Errorf("authentication required"))
			return
		}

		// Parse request body
		var req dto.RidesCancelRequestDto
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		// Validate reason
		if strings.TrimSpace(req.Reason) == "" {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("reason is required"))
			return
		}

		// Call service
		res, err := rh.ridesService.CancelRide(rideID, req.Reason, passengerID)
		if err != nil {
			statusCode := http.StatusInternalServerError
			if strings.Contains(err.Error(), "not found") {
				statusCode = http.StatusNotFound
			} else if strings.Contains(err.Error(), "access denied") {
				statusCode = http.StatusForbidden
			} else if strings.Contains(err.Error(), "cannot be cancelled") {
				statusCode = http.StatusConflict
			}
			jsonError(w, statusCode, err)
			return
		}

		jsonResponse(w, http.StatusOK, res)
	}
}

// Удалите функцию getPassengerIDFromContext, так как она больше не нужна