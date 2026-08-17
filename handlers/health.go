package handlers

import (
	"encoding/json"
	"net/http"
	"stock-advisor-site-golang/services"
	"time"
)

var serverStartTime = time.Now()

type HealthHandler struct {
	svc *services.MarketDataService
}

func NewHealthHandler(svc *services.MarketDataService) *HealthHandler {
	return &HealthHandler{svc: svc}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	info := h.svc.GetProviderInfo()
	warnings, _ := info["warnings"].([]string)
	status := "ok"
	if len(warnings) > 0 {
		status = "warn"
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":         status,
		"service":        "stock-advisor-site",
		"now":            time.Now().UTC().Format(time.RFC3339),
		"uptimeSeconds":  int(time.Since(serverStartTime).Seconds()),
		"provider":       info,
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
