package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"geofence-backend/internal/response"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ViolationHandler struct {
	pool *pgxpool.Pool
}

func NewViolationHandler(pool *pgxpool.Pool) *ViolationHandler {
	return &ViolationHandler{pool: pool}
}

type violationItem struct {
	ID            string    `json:"id"`
	VehicleID     string    `json:"vehicle_id"`
	VehicleNumber string    `json:"vehicle_number"`
	GeofenceID    string    `json:"geofence_id"`
	GeofenceName  string    `json:"geofence_name"`
	EventType     string    `json:"event_type"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	Timestamp     time.Time `json:"timestamp"`
}

// parseOptionalTime returns nil when the value is empty.
func parseOptionalTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// History handles GET /violations/history.
func (h *ViolationHandler) History(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	query := r.URL.Query()

	vehicleID := strings.TrimSpace(query.Get("vehicle_id"))
	geofenceID := strings.TrimSpace(query.Get("geofence_id"))

	startDate, err := parseOptionalTime(query.Get("start_date"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, start,
			"start_date must be in ISO 8601 format, for example 2025-01-01T00:00:00Z")
		return
	}
	endDate, err := parseOptionalTime(query.Get("end_date"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, start,
			"end_date must be in ISO 8601 format, for example 2025-01-31T23:59:59Z")
		return
	}

	limit := 50
	if value := strings.TrimSpace(query.Get("limit")); value != "" {
		limit, err = strconv.Atoi(value)
		if err != nil || limit < 1 {
			response.Error(w, http.StatusBadRequest, start, "limit must be a positive number")
			return
		}
		if limit > 500 {
			limit = 500
		}
	}

	offset := 0
	if value := strings.TrimSpace(query.Get("offset")); value != "" {
		offset, err = strconv.Atoi(value)
		if err != nil || offset < 0 {
			response.Error(w, http.StatusBadRequest, start, "offset must be zero or a positive number")
			return
		}
	}

	const filters = `
		WHERE ($1 = '' OR vi.vehicle_id = $1)
		  AND ($2 = '' OR vi.geofence_id = $2)
		  AND ($3::timestamptz IS NULL OR vi.occurred_at >= $3)
		  AND ($4::timestamptz IS NULL OR vi.occurred_at <= $4)`

	var totalCount int
	err = h.pool.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM violations vi`+filters,
		vehicleID, geofenceID, startDate, endDate).Scan(&totalCount)
	if err != nil {
		log.Printf("count violations failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not fetch history")
		return
	}

	rows, err := h.pool.Query(r.Context(),
		`SELECT vi.id, vi.vehicle_id, v.vehicle_number, vi.geofence_id, g.name,
		        vi.event_type, vi.latitude, vi.longitude, vi.occurred_at
		 FROM violations vi
		 JOIN vehicles v ON v.id = vi.vehicle_id
		 JOIN geofences g ON g.id = vi.geofence_id`+filters+`
		 ORDER BY vi.occurred_at DESC
		 LIMIT $5 OFFSET $6`,
		vehicleID, geofenceID, startDate, endDate, limit, offset)
	if err != nil {
		log.Printf("list violations failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not fetch history")
		return
	}
	defer rows.Close()

	violations := []violationItem{}
	for rows.Next() {
		var item violationItem
		if err := rows.Scan(&item.ID, &item.VehicleID, &item.VehicleNumber, &item.GeofenceID,
			&item.GeofenceName, &item.EventType, &item.Latitude, &item.Longitude,
			&item.Timestamp); err != nil {
			log.Printf("scan violation failed: %v", err)
			response.Error(w, http.StatusInternalServerError, start, "could not read history")
			return
		}
		violations = append(violations, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("read violations failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not read history")
		return
	}

	response.JSON(w, http.StatusOK, start, map[string]any{
		"violations":  violations,
		"total_count": totalCount,
	})
}