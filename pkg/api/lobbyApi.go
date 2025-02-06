package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/slinarji/go-geo-server/pkg/game"
	"github.com/slinarji/go-geo-server/pkg/websocket"
)

func ServeGetLobby(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(game.LobbyMap)
	slog.Debug("Sent lobby list")
}

func ServeCreateLobby(w http.ResponseWriter, r *http.Request) {
	// w.Header().Set("Content-Type", "application/json; charset=utf-8")

	var lobbyConf game.LobbyConf
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
	json.NewEncoder(w).Encode(game.CreateLobby(lobbyConf))

	slog.Info("Created new lobby")
}

func ServeDeleteLobby(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")
	slog.Warn("!! Delete lobby NOT implemented!!", "lobbyId", id)
	// delete(lobby.LobbyMap, id)

	w.WriteHeader(http.StatusOK)
}

func ServeLobbySocket(w http.ResponseWriter, r *http.Request) {
	// Added query parameter reader for id of lobby
	lobbyID := r.URL.Query().Get("id")

	ctx := r.Context()
	uid := ctx.Value("uid").(string)
	displayName := ctx.Value("displayname").(string)
	slog.Debug("WebSocket Endpoint Hit", "lobby ID", lobbyID, "uid", uid, " name", displayName)

	// only connect to ws if lobby exists
	if lobby, ok := game.LobbyMap[lobbyID]; ok {
		// check if player is already in lobby
		if player, ok := lobby.PlayerMap[uid]; ok {
			if player.Connected {
				slog.Error("Player already connected")
				w.WriteHeader(http.StatusConflict)
				return
			}
		} else if lobby.CurrentRound != 0 {
			slog.Error("Game in progres")
			w.WriteHeader(http.StatusConflict)
			return
		}

		conn, err := websocket.Upgrade(w, r)
		if err != nil {
			fmt.Fprintf(w, "%+v\n", err)
			return
		}

		game.AddPlayerToLobby(uid, displayName, lobbyID, conn)

	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}
