package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"geofence-backend/internal/ids"
	"geofence-backend/internal/response"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GeofenceHandler struct {
	pool *pgxpool.Pool
}

func NewGeofenceHandler(pool *pgxpool.Pool) *GeofenceHandler {
	return &GeofenceHandler{pool: pool}
}

var validCategories = map[string]bool{
	"delivery_zone":   true,
	"restricted_zone": true,
	"toll_zone":       true,
	"customer_area":   true,
}

type createGeofenceRequest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Coordinates [][]float64 `json:"coordinates"`
	Category    string      `json:"category"`
}

// validateCoordinates returns an error message, or an empty string if all is well.
func validateCoordinates(coords [][]float64) string {
	if len(coords) < 4 {
		return "coordinates must have at least 4 points (3 unique points and a closing point)"
	}
	for i, p := range coords {
		if len(p) != 2 {
			return fmt.Sprintf("point %d must be a [latitude, longitude] pair", i+1)
		}
		if p[0] < -90 || p[0] > 90 {
			return fmt.Sprintf("point %d: latitude must be between -90 and 90", i+1)
		}
		if p[1] < -180 || p[1] > 180 {
			return fmt.Sprintf("point %d: longitude must be between -180 and 180", i+1)
		}
	}
	first, last := coords[0], coords[len(coords)-1]
	if first[0] != last[0] || first[1] != last[1] {
		return "first and last coordinates must be identical (closed polygon)"
	}
	return ""
}

// toWKT turns [lat, lng] pairs into a PostGIS polygon, which expects longitude first.
func toWKT(coords [][]float64) string {
	parts := make([]string, len(coords))
	for i, p := range coords {
		lat := strconv.FormatFloat(p[0], 'f', -1, 64)
		lng := strconv.FormatFloat(p[1], 'f', -1, 64)
		parts[i] = lng + " " + lat
	}
	return "POLYGON((" + strings.Join(parts, ",") + "))"
}

// Create handles POST /geofences.
func (h *GeofenceHandler) Create(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req createGeofenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, start, "invalid JSON body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.Category = strings.TrimSpace(req.Category)

	if req.Name == "" {
		response.Error(w, http.StatusBadRequest, start, "name is required")
		return
	}
	if !validCategories[req.Category] {
		response.Error(w, http.StatusBadRequest, start,
			"category must be one of: delivery_zone, restricted_zone, toll_zone, customer_area")
		return
	}
	if msg := validateCoordinates(req.Coordinates); msg != "" {
		response.Error(w, http.StatusBadRequest, start, msg)
		return
	}

	wkt := toWKT(req.Coordinates)

	var isValid bool
	err := h.pool.QueryRow(r.Context(),
		`SELECT ST_IsValid(ST_GeomFromText($1, 4326))`, wkt).Scan(&isValid)
	if err != nil {
		log.Printf("polygon check failed: %v", err)
		response.Error(w, http.StatusBadRequest, start, "coordinates do not form a valid polygon")
		return
	}
	if !isValid {
		response.Error(w, http.StatusBadRequest, start,
			"polygon is not valid (for example, its edges may cross each other)")
		return
	}

	id := ids.New("geo")
	_, err = h.pool.Exec(r.Context(),
		`INSERT INTO geofences (id, name, description, category, boundary)
		 VALUES ($1, $2, $3, $4, ST_GeomFromText($5, 4326))`,
		id, req.Name, req.Description, req.Category, wkt)
	if err != nil {
		log.Printf("create geofence failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not create geofence")
		return
	}

	response.JSON(w, http.StatusCreated, start, map[string]any{
		"id":     id,
		"name":   req.Name,
		"status": "active",
	})
}