package public

import (
	"net/http"

	"github.com/gorilla/pat"
)

// Register will register public route mappings for paths handled by this
// package and its handlers.
func Register(r *pat.Router) {
	r.Get("/healthcheck", getHealthCheck).Name("get-healthcheck")
}

// HealthCheck returns the health of the application.
func getHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
