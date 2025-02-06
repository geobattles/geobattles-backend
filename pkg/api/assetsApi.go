package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/slinarji/go-geo-server/pkg/logic"
)

func ServeCountryList(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(logic.CountryList)
	slog.Debug("Sent country list")
}
