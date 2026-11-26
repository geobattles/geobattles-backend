package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/geobattles/geobattles-backend/pkg/game"
	"github.com/geobattles/geobattles-backend/pkg/websocket"
)

func ServeGetLobby(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(game.ListLobbies())
	slog.Debug("Sent lobby list")
}

func ServeLobbyDirectory(w http.ResponseWriter, r *http.Request) {
	caller, _ := r.Context().Value(UidKey).(string)
	JSON(w, http.StatusOK, game.LobbyDirectory(caller))
}

func ServeFindLobby(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	row, ok := game.FindLobbySummary(mux.Vars(r)["code"])
	if !ok {
		JSON(w, http.StatusNotFound, game.AdmissionStatus{Code: "LOBBY_NOT_FOUND"})
		return
	}
	JSON(w, http.StatusOK, row)
}

func ServeCreateLobby(w http.ResponseWriter, r *http.Request) {
	uid, ok := r.Context().Value(UidKey).(string)
	if !ok || uid == "" {
		ERROR(w, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}

	var lobbyConf game.LobbyConf
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Warn("Invalid lobby request body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err = json.Unmarshal(reqBody, &lobbyConf); err != nil {
		slog.Warn("Invalid lobby configuration JSON")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	lobby, err := game.CreateLobby(lobbyConf, uid)
	if err != nil {
		ERROR(w, http.StatusBadRequest, err)
		return
	}
	JSON(w, http.StatusOK, lobby)

	slog.Info("Created new lobby")
}

func ServeDeleteLobby(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")
	slog.Warn("!! Delete lobby NOT implemented!!", "lobbyId", id)

	w.WriteHeader(http.StatusOK)
}

// ServeLobbyAdmission reports only the caller's eligibility. The socket
// handshake still reserves and rechecks capacity and the exact membership.
func ServeLobbyAdmission(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	uid, ok := r.Context().Value(UidKey).(string)
	if !ok || uid == "" {
		ERROR(w, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}
	query := r.URL.Query()
	JSON(w, http.StatusOK, game.CheckAdmission(query.Get("id"), uid, query.Get("spectate") == "1"))
}

func ServeLobbySocket(w http.ResponseWriter, r *http.Request) {
	// Added query parameter reader for id of lobby
	query := r.URL.Query()

	ctx := r.Context()
	uid, hasID := ctx.Value(UidKey).(string)
	displayName, hasName := ctx.Value(DisplayNameKey).(string)
	if !hasID || uid == "" || !hasName {
		ERROR(w, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}

	admission, err := game.ReserveAdmission(query.Get("id"), uid, displayName, query.Get("spectate") == "1")
	if err != nil {
		status := http.StatusConflict
		if err.Error() == "LOBBY_NOT_FOUND" {
			status = http.StatusBadRequest
		}
		ERROR(w, status, err)
		return
	}
	defer admission.Cancel() // failed upgrade/expired reservation rollback
	conn, err := websocket.Upgrade(w, r)
	if err != nil {
		return // Upgrade already wrote the protocol response.
	}
	admission.Join(conn)
}
