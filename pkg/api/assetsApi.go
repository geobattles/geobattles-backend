package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/slinarji/go-geo-server/pkg/logic"
)

func ServeCountryList(w http.ResponseWriter, r *http.Request) {
	// w.Header().Set("Access-Control-Allow-Origin", "*")
	// w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	// w.Header().Set("Access-Control-Allow-Headers", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(logic.CountryList)
	slog.Info("sent country list")
}
