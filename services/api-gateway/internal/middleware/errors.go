package middleware

import (
	"encoding/json"
	"net/http"
)

// writeJSONError writes a JSON error response with the correct Content-Type header.
// Using http.Error would set Content-Type: text/plain, which breaks JSON clients.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
}
