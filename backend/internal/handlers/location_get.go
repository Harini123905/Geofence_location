package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"geofence-backend/internal/response"

	"github.com/jackc/pgx/v5"
)

// Get handles GET /vehicles/location/{vehicle_id}.
func (h *LocationHandler) Get(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	vehicleID := strings.TrimSpace(r.PathValue("vehicle_id"))

	var vehicleNumber string
	err := h.pool.QueryRow(ctx,
		`SELECT vehicle_number FROM vehicles WHERE id = $1`, vehicleID).Scan(&vehicleNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		response.Error(w, http.StatusNotFound, start, "vehicle not found")
		return
	}
	if err != nil {
		log.Printf("check vehicle failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not check vehicle")
		return
	}

	var lat, lng float64
	var recordedAt time.Time
	err = h.pool.QueryRow(ctx,
		`SELECT ST_Y(location), ST_X(location), recorded_at
		 FROM vehicle_locations
		 WHERE vehicle_id = $1
		 ORDER BY recorded_at DESC
		 LIMIT 1`, vehicleID).Scan(&lat, &lng, &recordedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		response.JSON(w, http.StatusOK, start, map[string]any{
			"vehicle_id":        vehicleID,
			"vehicle_number":    vehicleNumber,
			"current_location":  nil,
			"current_geofences": []map[string]any{},
		})
		return
	}
	if err != nil {
		log.Printf("read location failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not read location")
		return
	}

	rows, err := h.pool.Query(ctx,
		`SELECT id, name, category FROM geofences
		 WHERE ST_Covers(boundary, ST_SetSRID(ST_MakePoint($1, $2), 4326))`, lng, lat)
	if err != nil {
		log.Printf("find geofences failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not check geofences")
		return
	}
	current, err := collectGeofences(rows)
	if err != nil {
		log.Printf("read geofences failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not check geofences")
		return
	}

	currentList := []map[string]any{}
	for _, g := range current {
		currentList = append(currentList, map[string]any{
			"geofence_id":   g.ID,
			"geofence_name": g.Name,
			"category":      g.Category,
		})
	}

	response.JSON(w, http.StatusOK, start, map[string]any{
		"vehicle_id":     vehicleID,
		"vehicle_number": vehicleNumber,
		"current_location": map[string]any{
			"latitude":  lat,
			"longitude": lng,
			"timestamp": recordedAt,
		},
		"current_geofences": currentList,
	})
}