package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"geofence-backend/internal/ids"
	"geofence-backend/internal/response"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertHandler struct {
	pool *pgxpool.Pool
}

func NewAlertHandler(pool *pgxpool.Pool) *AlertHandler {
	return &AlertHandler{pool: pool}
}

type configureAlertRequest struct {
	GeofenceID string  `json:"geofence_id"`
	VehicleID  *string `json:"vehicle_id"`
	EventType  string  `json:"event_type"`
}

var validEventTypes = map[string]bool{
	"entry": true,
	"exit":  true,
	"both":  true,
}

// Configure handles POST /alerts/configure.
func (h *AlertHandler) Configure(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req configureAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, start, "invalid JSON body")
		return
	}

	req.GeofenceID = strings.TrimSpace(req.GeofenceID)
	req.EventType = strings.TrimSpace(req.EventType)

	// An empty vehicle_id means the rule applies to all vehicles.
	if req.VehicleID != nil {
		trimmed := strings.TrimSpace(*req.VehicleID)
		if trimmed == "" {
			req.VehicleID = nil
		} else {
			req.VehicleID = &trimmed
		}
	}

	if req.GeofenceID == "" {
		response.Error(w, http.StatusBadRequest, start, "geofence_id is required")
		return
	}
	if !validEventTypes[req.EventType] {
		response.Error(w, http.StatusBadRequest, start, "event_type must be one of: entry, exit, both")
		return
	}

	var geofenceExists bool
	err := h.pool.QueryRow(r.Context(),
		`SELECT EXISTS (SELECT 1 FROM geofences WHERE id = $1)`, req.GeofenceID).Scan(&geofenceExists)
	if err != nil {
		log.Printf("check geofence failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not check geofence")
		return
	}
	if !geofenceExists {
		response.Error(w, http.StatusNotFound, start, "geofence not found")
		return
	}

	if req.VehicleID != nil {
		var vehicleExists bool
		err := h.pool.QueryRow(r.Context(),
			`SELECT EXISTS (SELECT 1 FROM vehicles WHERE id = $1)`, *req.VehicleID).Scan(&vehicleExists)
		if err != nil {
			log.Printf("check vehicle failed: %v", err)
			response.Error(w, http.StatusInternalServerError, start, "could not check vehicle")
			return
		}
		if !vehicleExists {
			response.Error(w, http.StatusNotFound, start, "vehicle not found")
			return
		}
	}

	id := ids.New("alert")
	_, err = h.pool.Exec(r.Context(),
		`INSERT INTO alert_rules (id, geofence_id, vehicle_id, event_type)
		 VALUES ($1, $2, $3, $4)`,
		id, req.GeofenceID, req.VehicleID, req.EventType)
	if err != nil {
		log.Printf("create alert rule failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not create alert rule")
		return
	}

	response.JSON(w, http.StatusCreated, start, map[string]any{
		"alert_id":    id,
		"geofence_id": req.GeofenceID,
		"vehicle_id":  req.VehicleID,
		"event_type":  req.EventType,
		"status":      "active",
	})
}