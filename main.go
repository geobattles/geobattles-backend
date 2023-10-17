package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/slinarji/go-geo-server/pkg/lobby"
	"github.com/slinarji/go-geo-server/pkg/logic"
	"github.com/slinarji/go-geo-server/pkg/websocket"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
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
		fmt.Println("Sent lobby list")
	case http.MethodPost:
		var lobbyConf logic.LobbyConf
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err = json.Unmarshal(reqBody, &lobbyConf); err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(lobby.CreateLobby(lobbyConf))
	// TODO: only allow admin? to delete lobby
	case http.MethodDelete:
		delete(lobby.LobbyMap, id)
	}
}

func serveCountryList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(logic.CountryList)
	fmt.Println("sent country list")
}

func serveLobbySocket(pool *websocket.Pool, w http.ResponseWriter, r *http.Request) {
	// Added query parameter reader for id of lobby
	lobbyID := r.URL.Query().Get("id")
	userName := r.URL.Query().Get("name")
	fmt.Println("WebSocket Endpoint Hit, room ID: ", lobbyID, " name: ", userName)
	// only connect to ws if lobby exists
	if _, ok := lobby.LobbyMap[lobbyID]; ok {
		if lobby.LobbyMap[lobbyID].CurrentRound != 0 {
			fmt.Println("ERR: Game in progres")
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
		fmt.Println("lobbyList: ", lobby.LobbyMap)

		go client.Read()
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func setupRoutes(r *mux.Router) {
	pool := websocket.NewPool()
	go pool.Start()
	r.HandleFunc("/lobbySocket", func(w http.ResponseWriter, r *http.Request) {
		serveLobbySocket(pool, w, r)
	})
	r.HandleFunc("/lobby", serveLobby)
	r.HandleFunc("/countryList", serveCountryList)
}

func main() {
	//set seed for rand function
	rand.Seed(time.Now().UnixNano())
	// try to read .env file
	// in docker we just use ENV variables and this WILL throw error
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}
	router := mux.NewRouter()
	setupRoutes(router)
	logic.InitCountryDB()
	http.ListenAndServe("0.0.0.0:8080", router)
}
