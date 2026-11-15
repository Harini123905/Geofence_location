package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"geofence-backend/internal/database"
	"geofence-backend/internal/handlers"
	"geofence-backend/internal/middleware"
	"geofence-backend/internal/realtime"
	"geofence-backend/internal/response"
)

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	databaseURL := getEnv(
		"DATABASE_URL",
		"postgres://geofence:geofence_pass@localhost:5433/geofence_db?sslmode=disable",
	)

	pool, err := database.Connect(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to PostgreSQL")

	hub := realtime.NewHub()

	vehicleHandler := handlers.NewVehicleHandler(pool)
	geofenceHandler := handlers.NewGeofenceHandler(pool)
	alertHandler := handlers.NewAlertHandler(pool)
	locationHandler := handlers.NewLocationHandler(pool, hub)
	violationHandler := handlers.NewViolationHandler(pool)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		status := "ok"
		httpStatus := http.StatusOK
		if err := pool.Ping(r.Context()); err != nil {
			status = "database unavailable"
			httpStatus = http.StatusServiceUnavailable
		}
		response.JSON(w, httpStatus, start, map[string]any{"status": status})
	})

	mux.HandleFunc("POST /vehicles", vehicleHandler.Create)
	mux.HandleFunc("GET /vehicles", vehicleHandler.List)
	mux.HandleFunc("POST /vehicles/location", locationHandler.Update)
	mux.HandleFunc("GET /vehicles/location/{vehicle_id}", locationHandler.Get)

	mux.HandleFunc("POST /geofences", geofenceHandler.Create)
	mux.HandleFunc("GET /geofences", geofenceHandler.List)

	mux.HandleFunc("POST /alerts/configure", alertHandler.Configure)
	mux.HandleFunc("GET /alerts", alertHandler.List)

	mux.HandleFunc("GET /violations/history", violationHandler.History)

	mux.HandleFunc("GET /ws/alerts", hub.ServeWS)

	port := getEnv("PORT", "8080")
	log.Printf("Server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, middleware.CORS(mux)))
}