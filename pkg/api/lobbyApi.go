package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/slinarji/go-geo-server/pkg/lobby"
	"github.com/slinarji/go-geo-server/pkg/logic"
)

func ServeGetLobby(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(lobby.LobbyMap)
	slog.Info("Sent lobby list")
}

func ServeCreateLobby(w http.ResponseWriter, r *http.Request) {
	// w.Header().Set("Content-Type", "application/json; charset=utf-8")

	var lobbyConf logic.LobbyConf
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err = json.Unmarshal(reqBody, &lobbyConf); err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(lobby.CreateLobby(lobbyConf))

	slog.Info("Created new lobby")
}

func ServeDeleteLobby(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")
	slog.Info("!!NOT IMPLEMENTED!! Deleted", "lobbyId", id)
	// delete(lobby.LobbyMap, id)

	w.WriteHeader(http.StatusOK)
}
