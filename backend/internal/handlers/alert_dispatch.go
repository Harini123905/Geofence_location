package handlers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"geofence-backend/internal/ids"
	"geofence-backend/internal/realtime"

	"github.com/jackc/pgx/v5/pgxpool"
)

type alertDispatcher struct {
	pool *pgxpool.Pool
	hub  *realtime.Hub
}

type alertVehicle struct {
	VehicleID     string `json:"vehicle_id"`
	VehicleNumber string `json:"vehicle_number"`
	DriverName    string `json:"driver_name"`
}

type alertGeofence struct {
	GeofenceID   string `json:"geofence_id"`
	GeofenceName string `json:"geofence_name"`
	Category     string `json:"category"`
}

type alertLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type alertMessage struct {
	EventID   string        `json:"event_id"`
	EventType string        `json:"event_type"`
	Timestamp string        `json:"timestamp"`
	Vehicle   alertVehicle  `json:"vehicle"`
	Geofence  alertGeofence `json:"geofence"`
	Location  alertLocation `json:"location"`
}

// dispatch runs in the background. It must not use the request context,
// because that is cancelled as soon as the HTTP response is sent.
func (d *alertDispatcher) dispatch(vehicleID string, lat, lng float64, recordedAt time.Time, events []locationEvent) {
	if len(events) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var vehicleNumber, driverName string
	err := d.pool.QueryRow(ctx,
		`SELECT vehicle_number, driver_name FROM vehicles WHERE id = $1`, vehicleID).
		Scan(&vehicleNumber, &driverName)
	if err != nil {
		log.Printf("alert: could not load vehicle %s: %v", vehicleID, err)
		return
	}

	for _, ev := range events {
		var matched bool
		err := d.pool.QueryRow(ctx,
			`SELECT EXISTS (
				SELECT 1 FROM alert_rules
				WHERE geofence_id = $1
				  AND (vehicle_id IS NULL OR vehicle_id = $2)
				  AND event_type IN ($3, 'both'))`,
			ev.Geofence.ID, vehicleID, ev.Type).Scan(&matched)
		if err != nil {
			log.Printf("alert: rule check failed: %v", err)
			continue
		}
		if !matched {
			continue
		}

		message := alertMessage{
			EventID:   ids.New("evt"),
			EventType: ev.Type,
			Timestamp: recordedAt.UTC().Format(time.RFC3339),
			Vehicle: alertVehicle{
				VehicleID:     vehicleID,
				VehicleNumber: vehicleNumber,
				DriverName:    driverName,
			},
			Geofence: alertGeofence{
				GeofenceID:   ev.Geofence.ID,
				GeofenceName: ev.Geofence.Name,
				Category:     ev.Geofence.Category,
			},
			Location: alertLocation{Latitude: lat, Longitude: lng},
		}

		payload, err := json.Marshal(message)
		if err != nil {
			log.Printf("alert: could not encode message: %v", err)
			continue
		}
		d.hub.Broadcast(payload)
		log.Printf("ALERT sent: %s %s %s", vehicleNumber, ev.Type, ev.Geofence.Name)
	}
}