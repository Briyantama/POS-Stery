package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/pos-stery/pos-stery/services/api-gateway/internal/clients"
)

// renderJSON writes v as JSON with the given status code.
func renderJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("renderJSON encode error: %v", err)
	}
}

// renderError writes an error response. If err wraps a *clients.GrpcError the
// HTTP status is derived from the gRPC code; otherwise 500 is used.
func renderError(w http.ResponseWriter, err error) {
	var ge *clients.GrpcError
	if errors.As(err, &ge) {
		renderJSON(w, ge.HTTPStatus, map[string]string{"message": ge.Message})
		return
	}
	renderJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal server error"})
}

// validationError writes a 422 response in Laravel-compatible format.
func validationError(w http.ResponseWriter, field, msg string) {
	renderJSON(w, http.StatusUnprocessableEntity, map[string]any{
		"message": "Validation failed.",
		"errors":  map[string][]string{field: {msg}},
	})
}
