package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

const readinessTimeout = time.Second

// Diagnostics holds startup state and public build identity, captured before serving.
// It does not read runtime configuration or expose database/provider errors.
type Diagnostics struct {
	database    *sql.DB
	initialized bool
	version     string
}

func NewDiagnostics(database *sql.DB, initialized bool, version string) *Diagnostics {
	return &Diagnostics{database: database, initialized: initialized, version: version}
}

func (d *Diagnostics) ServeHealth(w http.ResponseWriter, r *http.Request) {
	if !prepareDiagnostic(w, r) {
		return
	}
	writeDiagnostic(w, r, http.StatusOK, diagnosticStatus{Status: "ok"})
}

func (d *Diagnostics) ServeReady(w http.ResponseWriter, r *http.Request) {
	if !prepareDiagnostic(w, r) {
		return
	}
	status := http.StatusServiceUnavailable
	body := diagnosticStatus{Status: "unavailable"}
	if d.initialized && d.database != nil {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()
		if err := d.database.PingContext(ctx); err == nil {
			status = http.StatusOK
			body.Status = "ready"
		}
	}
	writeDiagnostic(w, r, status, body)
}

func (d *Diagnostics) ServeVersion(w http.ResponseWriter, r *http.Request) {
	if !prepareDiagnostic(w, r) {
		return
	}
	// Keep the public build contract allowlisted; never serialize build settings,
	// environment variables, hostnames, paths, or database connection information.
	writeDiagnostic(w, r, http.StatusOK, struct {
		Version string `json:"version"`
	}{Version: d.version})
}

type diagnosticStatus struct {
	Status string `json:"status"`
}

func prepareDiagnostic(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		return true
	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, HEAD, OPTIONS")
		writeDiagnostic(w, r, http.StatusMethodNotAllowed, diagnosticStatus{Status: "method_not_allowed"})
	}
	return false
}

func writeDiagnostic(w http.ResponseWriter, r *http.Request, status int, body any) {
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		_ = json.NewEncoder(w).Encode(body)
	}
}
