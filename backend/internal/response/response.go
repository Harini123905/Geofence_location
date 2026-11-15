package response

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// JSON writes a JSON response and adds time_ns, the time taken since start.
func JSON(w http.ResponseWriter, status int, start time.Time, payload map[string]any) {
	payload["time_ns"] = strconv.FormatInt(time.Since(start).Nanoseconds(), 10)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

// Error writes an error message in the same format, with time_ns included.
func Error(w http.ResponseWriter, status int, start time.Time, message string) {
	JSON(w, status, start, map[string]any{"error": message})
}