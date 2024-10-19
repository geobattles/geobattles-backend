package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	"github.com/slinarji/go-geo-server/pkg/api"
	"github.com/slinarji/go-geo-server/pkg/lobby"
	"github.com/slinarji/go-geo-server/pkg/logic"
	"github.com/slinarji/go-geo-server/pkg/reverse"
	"github.com/slinarji/go-geo-server/pkg/websocket"
)

func serveLobby(w http.ResponseWriter, r *http.Request) {
	// deal with CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	if r.Method == "OPTIONS" {
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	id := r.URL.Query().Get("id")
	switch r.Method {
	case http.MethodGet:
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(lobby.LobbyMap)
		slog.Info("Sent lobby list")
	case http.MethodPost:
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
	// TODO: only allow admin? to delete lobby
	case http.MethodDelete:
		delete(lobby.LobbyMap, id)
	}
}

func serveLobbySocket(pool *websocket.Pool, w http.ResponseWriter, r *http.Request) {
	// Added query parameter reader for id of lobby
	lobbyID := r.URL.Query().Get("id")
	userName := r.URL.Query().Get("name")
	slog.Info("WebSocket Endpoint Hit, room ID: ", lobbyID, " name: ", userName)
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
			Name: userName,
			ID:   logic.GenerateRndID(8),
		}
		lobby.AddPlayerToLobby(client.ID, client.Name, lobbyID)

		pool.Register <- client
		slog.Info("lobbyList: ", lobby.LobbyMap)

		go client.Read()
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func setupRoutes(r *mux.Router) {
	r.HandleFunc("/register", api.RegisterUser).Methods("POST")       // register user
	r.HandleFunc("/login", api.LoginUser).Methods("POST")             // login user
	r.HandleFunc("/countryList", api.ServeCountryList).Methods("GET") // send list of available countries
	r.HandleFunc("/lobby", serveLobby)
	pool := websocket.NewPool()
	go pool.Start()
	r.HandleFunc("/lobbySocket", func(w http.ResponseWriter, r *http.Request) {
		serveLobbySocket(pool, w, r)
	})
}

func init() {
	// try to read .env file
	// in docker we just use ENV variables and this WILL throw an error
	err := godotenv.Load()
	if err != nil {
		slog.Info("Error loading .env file")
	}

	logic.InitCountryDB()
	err2 := reverse.InitReverse()
	if err2 != nil {
		slog.Error(err2.Error())
	}
}

func main() {
	router := mux.NewRouter()
	setupRoutes(router)

	slog.Info("Server is ready")
	err := http.ListenAndServe("0.0.0.0:8080", router)
	if err != nil {
		slog.Error(err.Error())
	}

}
