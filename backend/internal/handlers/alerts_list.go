package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"

	"geofence-backend/internal/response"
)

type alertItem struct {
	AlertID       string    `json:"alert_id"`
	GeofenceID    string    `json:"geofence_id"`
	GeofenceName  string    `json:"geofence_name"`
	VehicleID     *string   `json:"vehicle_id"`
	VehicleNumber *string   `json:"vehicle_number"`
	EventType     string    `json:"event_type"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// List handles GET /alerts, with optional ?geofence_id= and ?vehicle_id= filters.
func (h *AlertHandler) List(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	geofenceID := strings.TrimSpace(r.URL.Query().Get("geofence_id"))
	vehicleID := strings.TrimSpace(r.URL.Query().Get("vehicle_id"))

	rows, err := h.pool.Query(r.Context(),
		`SELECT ar.id, ar.geofence_id, g.name, ar.vehicle_id, v.vehicle_number,
		        ar.event_type, ar.status, ar.created_at
		 FROM alert_rules ar
		 JOIN geofences g ON g.id = ar.geofence_id
		 LEFT JOIN vehicles v ON v.id = ar.vehicle_id
		 WHERE ($1 = '' OR ar.geofence_id = $1)
		   AND ($2 = '' OR ar.vehicle_id = $2)
		 ORDER BY ar.created_at DESC`, geofenceID, vehicleID)
	if err != nil {
		log.Printf("list alerts failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not fetch alerts")
		return
	}
	defer rows.Close()

	alerts := []alertItem{}
	for rows.Next() {
		var a alertItem
		if err := rows.Scan(&a.AlertID, &a.GeofenceID, &a.GeofenceName, &a.VehicleID,
			&a.VehicleNumber, &a.EventType, &a.Status, &a.CreatedAt); err != nil {
			log.Printf("scan alert failed: %v", err)
			response.Error(w, http.StatusInternalServerError, start, "could not read alerts")
			return
		}
		alerts = append(alerts, a)
	}
	if err := rows.Err(); err != nil {
		log.Printf("read alerts failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not read alerts")
		return
	}

	response.JSON(w, http.StatusOK, start, map[string]any{"alerts": alerts})
}