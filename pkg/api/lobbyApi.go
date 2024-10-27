package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/slinarji/go-geo-server/pkg/lobby"
	"github.com/slinarji/go-geo-server/pkg/logic"
	"github.com/slinarji/go-geo-server/pkg/websocket"
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

var pool *websocket.Pool

func init() {
	pool = websocket.NewPool()
	go pool.Start()
}

func ServeLobbySocket(w http.ResponseWriter, r *http.Request) {
	// Added query parameter reader for id of lobby
	lobbyID := r.URL.Query().Get("id")

	ctx := r.Context()
	uid := ctx.Value("uid").(string)
	displayName := ctx.Value("displayname").(string)
	slog.Info("WebSocket Endpoint Hit, room ID: ", lobbyID, "uid : ", uid, " name: ", displayName)

	// only connect to ws if lobby exists
	if _, ok := lobby.LobbyMap[lobbyID]; ok {
		if lobby.LobbyMap[lobbyID].CurrentRound != 0 {
			slog.Error("ERR: Game in progres")
			w.WriteHeader(http.StatusConflict)
			return
		}

		conn, err := websocket.Upgrade(w, r)
		if err != nil {
			fmt.Fprintf(w, "%+v\n", err)
			return
		}

		client := &websocket.Client{
			Conn: conn,
			Pool: pool,
			Room: lobbyID,
			Name: displayName,
			ID:   uid,
		}
		lobby.AddPlayerToLobby(client.ID, client.Name, lobbyID)

		pool.Register <- client
		slog.Info("lobbyList: ", lobby.LobbyMap)

		go client.Read()
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}
