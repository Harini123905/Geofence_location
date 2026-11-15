package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"geofence-backend/internal/ids"
	"geofence-backend/internal/response"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VehicleHandler struct {
	pool *pgxpool.Pool
}

func NewVehicleHandler(pool *pgxpool.Pool) *VehicleHandler {
	return &VehicleHandler{pool: pool}
}

type createVehicleRequest struct {
	VehicleNumber string `json:"vehicle_number"`
	DriverName    string `json:"driver_name"`
	VehicleType   string `json:"vehicle_type"`
	Phone         string `json:"phone"`
}

// Create handles POST /vehicles.
func (h *VehicleHandler) Create(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req createVehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, start, "invalid JSON body")
		return
	}

	req.VehicleNumber = strings.TrimSpace(req.VehicleNumber)
	req.DriverName = strings.TrimSpace(req.DriverName)
	req.VehicleType = strings.TrimSpace(req.VehicleType)
	req.Phone = strings.TrimSpace(req.Phone)

	if req.VehicleNumber == "" || req.DriverName == "" || req.VehicleType == "" || req.Phone == "" {
		response.Error(w, http.StatusBadRequest, start,
			"vehicle_number, driver_name, vehicle_type and phone are required")
		return
	}

	id := ids.New("veh")
	_, err := h.pool.Exec(r.Context(),
		`INSERT INTO vehicles (id, vehicle_number, driver_name, vehicle_type, phone)
		 VALUES ($1, $2, $3, $4, $5)`,
		id, req.VehicleNumber, req.DriverName, req.VehicleType, req.Phone)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			response.Error(w, http.StatusConflict, start, "vehicle_number already exists")
			return
		}
		log.Printf("create vehicle failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not create vehicle")
		return
	}

	response.JSON(w, http.StatusCreated, start, map[string]any{
		"id":             id,
		"vehicle_number": req.VehicleNumber,
		"status":         "active",
	})
}

type vehicleItem struct {
	ID            string    `json:"id"`
	VehicleNumber string    `json:"vehicle_number"`
	DriverName    string    `json:"driver_name"`
	VehicleType   string    `json:"vehicle_type"`
	Phone         string    `json:"phone"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// List handles GET /vehicles.
func (h *VehicleHandler) List(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	rows, err := h.pool.Query(r.Context(),
		`SELECT id, vehicle_number, driver_name, vehicle_type, phone, status, created_at
		 FROM vehicles
		 ORDER BY created_at DESC`)
	if err != nil {
		log.Printf("list vehicles failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not fetch vehicles")
		return
	}
	defer rows.Close()

	vehicles := []vehicleItem{}
	for rows.Next() {
		var v vehicleItem
		if err := rows.Scan(&v.ID, &v.VehicleNumber, &v.DriverName,
			&v.VehicleType, &v.Phone, &v.Status, &v.CreatedAt); err != nil {
			log.Printf("scan vehicle failed: %v", err)
			response.Error(w, http.StatusInternalServerError, start, "could not read vehicles")
			return
		}
		vehicles = append(vehicles, v)
	}
	if err := rows.Err(); err != nil {
		log.Printf("read vehicles failed: %v", err)
		response.Error(w, http.StatusInternalServerError, start, "could not read vehicles")
		return
	}

	response.JSON(w, http.StatusOK, start, map[string]any{"vehicles": vehicles})
}