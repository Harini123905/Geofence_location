package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"geofence-backend/internal/ids"
	"geofence-backend/internal/realtime"
	"geofence-backend/internal/response"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LocationHandler struct {
	pool       *pgxpool.Pool
	dispatcher *alertDispatcher
}

func NewLocationHandler(pool *pgxpool.Pool, hub *realtime.Hub) *LocationHandler {
	return &LocationHandler{
		pool:       pool,
		dispatcher: &alertDispatcher{pool: pool, hub: hub},
	}
}

type updateLocationRequest struct {
	VehicleID string   `json:"vehicle_id"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	Timestamp string   `json:"timestamp"`
}

type geofenceRef struct {
	ID       string
	Name     string
	Category string
}

type locationEvent struct {
	Type     string
	Geofence geofenceRef
}

// collectGeofences reads all rows (id, name, category) and closes them.
func collectGeofences(rows pgx.Rows) ([]geofenceRef, error) {
	defer rows.Close()
	list := []geofenceRef{}
	for rows.Next() {
		var g geofenceRef
		if err := rows.Scan(&g.ID, &g.Name, &g.Category); err != nil {
			return nil, err
		}
		list = append(list, g)
	}
	return list, rows.Err()
}

// previousGeofenceIDs returns the geofences the vehicle was inside at its last update.
func previousGeofenceIDs(ctx context.Context, tx pgx.Tx, vehicleID string) (map[string]bool, error) {
	rows, err := tx.Query(ctx,
		`SELECT geofence_id FROM vehicle_geofence_state WHERE vehicle_id = $1`, vehicleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	previous := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		previous[id] = true
	}
	return previous, rows.Err()
}

// Update handles POST /vehicles/location.
func (h *LocationHandler) Update(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	var req updateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, start, "invalid JSON body")
		return
	}

	req.VehicleID = strings.TrimSpace(req.VehicleID)
	if req.VehicleID == "" {
		response.Error(w, http.StatusBadRequest, start, "vehicle_id is required")
		return
	}
	if req.Latitude == nil || req.Longitude == nil {
		response.Error(w, http.StatusBadRequest, start, "latitude and longitude are required")
		return
	}
	lat, lng := *req.Latitude, *req.Longitude
	if lat < -90 || lat > 90 {
		response.Error(w, http.StatusBadRequest, start, "latitude must be between -90 and 90")
		return
	}
	if lng < -180 || lng > 180 {
		response.Error(w, http.StatusBadRequest, start, "longitude must be between -180 and 180")
		return
	}
	recordedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(req.Timestamp))
	if err != nil {
		response.Error(w, http.StatusBadRequest, start,
			"timestamp must be in ISO 8601 format, for example 2025-01-15T10:35:00Z")
		return
	}

	var vehicleID string
	err = h.pool.QueryRow(ctx, `SELECT id FROM vehicles WHERE id = $1`, req.VehicleID).Scan(&vehicleID)
	if errors.Is(err, pgx.ErrNoRows) {
		response.Error(w, http.StatusNotFound, start, "vehicle not found")
		return
	}
	if err != nil {
		log.Printf("check vehicle failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not check vehicle")
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		log.Printf("begin transaction failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not update location")
		return
	}
	defer tx.Rollback(ctx)

	// 1. Save the location.
	_, err = tx.Exec(ctx,
		`INSERT INTO vehicle_locations (vehicle_id, location, recorded_at)
		 VALUES ($1, ST_SetSRID(ST_MakePoint($2, $3), 4326), $4)`,
		vehicleID, lng, lat, recordedAt)
	if err != nil {
		log.Printf("save location failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not save location")
		return
	}

	// 2. Which geofences contain this point? ST_Covers also counts the boundary as inside.
	rows, err := tx.Query(ctx,
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

	// 3. Compare with the previous state to find entries and exits.
	previous, err := previousGeofenceIDs(ctx, tx, vehicleID)
	if err != nil {
		log.Printf("read previous state failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not read previous state")
		return
	}

	currentIDs := map[string]bool{}
	events := []locationEvent{}
	for _, g := range current {
		currentIDs[g.ID] = true
		if !previous[g.ID] {
			events = append(events, locationEvent{Type: "entry", Geofence: g})
		}
	}

	exitedIDs := []string{}
	for id := range previous {
		if !currentIDs[id] {
			exitedIDs = append(exitedIDs, id)
		}
	}
	if len(exitedIDs) > 0 {
		rows, err := tx.Query(ctx,
			`SELECT id, name, category FROM geofences WHERE id = ANY($1)`, exitedIDs)
		if err != nil {
			log.Printf("find exited geofences failed: %v", err)
			response.Error(w, http.StatusInternalServerError, start, "could not check geofences")
			return
		}
		exited, err := collectGeofences(rows)
		if err != nil {
			log.Printf("read exited geofences failed: %v", err)
			response.Error(w, http.StatusInternalServerError, start, "could not check geofences")
			return
		}
		for _, g := range exited {
			events = append(events, locationEvent{Type: "exit", Geofence: g})
		}
	}

	// 4. Save the new state and the events.
	for _, ev := range events {
		if ev.Type == "entry" {
			_, err = tx.Exec(ctx,
				`INSERT INTO vehicle_geofence_state (vehicle_id, geofence_id, entered_at)
				 VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
				vehicleID, ev.Geofence.ID, recordedAt)
		} else {
			_, err = tx.Exec(ctx,
				`DELETE FROM vehicle_geofence_state WHERE vehicle_id = $1 AND geofence_id = $2`,
				vehicleID, ev.Geofence.ID)
		}
		if err != nil {
			log.Printf("update state failed: %v", err)
			response.Error(w, http.StatusInternalServerError, start, "could not update state")
			return
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO violations (id, vehicle_id, geofence_id, event_type, latitude, longitude, occurred_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			ids.New("viol"), vehicleID, ev.Geofence.ID, ev.Type, lat, lng, recordedAt)
		if err != nil {
			log.Printf("save event failed: %v", err)
			response.Error(w, http.StatusInternalServerError, start, "could not save event")
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("commit failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not save location")
		return
	}

	for _, ev := range events {
		log.Printf("EVENT: vehicle %s %s geofence %s", vehicleID, ev.Type, ev.Geofence.Name)
	}

	// Send live alerts in the background so the HTTP response is not delayed.
	go h.dispatcher.dispatch(vehicleID, lat, lng, recordedAt, events)

	currentList := []map[string]any{}
	for _, g := range current {
		currentList = append(currentList, map[string]any{
			"geofence_id":   g.ID,
			"geofence_name": g.Name,
			"status":        "inside",
		})
	}

	response.JSON(w, http.StatusOK, start, map[string]any{
		"vehicle_id":        vehicleID,
		"location_updated":  true,
		"current_geofences": currentList,
	})
}