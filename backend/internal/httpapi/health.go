package httpapi

import (
	"net/http"
)

// HandleHealthz returns a simple health check handler.
func HandleHealthz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
