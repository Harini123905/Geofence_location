package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"geofence-backend/internal/response"
)

type geofenceItem struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Coordinates [][2]float64 `json:"coordinates"`
	Category    string       `json:"category"`
	CreatedAt   time.Time    `json:"created_at"`
}

type geoJSONPolygon struct {
	Coordinates [][][]float64 `json:"coordinates"`
}

// toLatLng reads PostGIS GeoJSON (longitude first) and returns [latitude, longitude] pairs.
func toLatLng(geoJSON string) ([][2]float64, error) {
	var poly geoJSONPolygon
	if err := json.Unmarshal([]byte(geoJSON), &poly); err != nil {
		return nil, err
	}
	result := [][2]float64{}
	if len(poly.Coordinates) == 0 {
		return result, nil
	}
	for _, point := range poly.Coordinates[0] {
		if len(point) >= 2 {
			result = append(result, [2]float64{point[1], point[0]})
		}
	}
	return result, nil
}

// List handles GET /geofences, with an optional ?category= filter.
func (h *GeofenceHandler) List(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	category := strings.TrimSpace(r.URL.Query().Get("category"))
	if category != "" && !validCategories[category] {
		response.Error(w, http.StatusBadRequest, start,
			"category must be one of: delivery_zone, restricted_zone, toll_zone, customer_area")
		return
	}

	rows, err := h.pool.Query(r.Context(),
		`SELECT id, name, description, category, ST_AsGeoJSON(boundary), created_at
		 FROM geofences
		 WHERE ($1 = '' OR category = $1)
		 ORDER BY created_at DESC`, category)
	if err != nil {
		log.Printf("list geofences failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not fetch geofences")
		return
	}
	defer rows.Close()

	geofences := []geofenceItem{}
	for rows.Next() {
		var g geofenceItem
		var geoJSON string
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.Category, &geoJSON, &g.CreatedAt); err != nil {
			log.Printf("scan geofence failed: %v", err)
			response.Error(w, http.StatusInternalServerError, start, "could not read geofences")
			return
		}
		coords, err := toLatLng(geoJSON)
		if err != nil {
			log.Printf("parse geofence boundary failed: %v", err)
			response.Error(w, http.StatusInternalServerError, start, "could not read geofence boundary")
			return
		}
		g.Coordinates = coords
		geofences = append(geofences, g)
	}
	if err := rows.Err(); err != nil {
		log.Printf("read geofences failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not read geofences")
		return
	}

	response.JSON(w, http.StatusOK, start, map[string]any{"geofences": geofences})
}