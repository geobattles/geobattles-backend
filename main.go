package main

import (
	"encoding/json"
	"example/web-service-gin/pkg/lobby"
	"example/web-service-gin/pkg/logic"
	"example/web-service-gin/pkg/websocket"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

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
	w.WriteHeader(http.StatusOK)
	id := r.URL.Query().Get("id")
	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(lobby.LobbyMap)
		fmt.Println("Sent lobby list")
		//fmt.Println(runtime.NumGoroutine())
	case http.MethodPost:
		var newLobby *logic.Lobby
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		if err = json.Unmarshal(reqBody, &newLobby); err != nil {
			fmt.Println(err)
			return
		}

		json.NewEncoder(w).Encode(lobby.CreateLobby(newLobby))
	// TODO: only allow admin? to delete lobby
	case http.MethodDelete:
		delete(lobby.LobbyMap, id)
	}
}

func serveLobbySocket(pool *websocket.Pool, w http.ResponseWriter, r *http.Request) {
	// Added query parameter reader for id of lobby
	lobbyID := r.URL.Query().Get("id")
	userName := r.URL.Query().Get("name")
	fmt.Println("WebSocket Endpoint Hit, room ID: ", lobbyID, " name: ", userName)
	// only connect to ws if lobby exists
	if _, ok := lobby.LobbyMap[lobbyID]; ok {
		conn, err := websocket.Upgrade(w, r)
		if err != nil {
			fmt.Fprintf(w, "%+v\n", err)
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
	}
}

func setupRoutes(r *mux.Router) {
	pool := websocket.NewPool()
	go pool.Start()
	r.HandleFunc("/lobbySocket", func(w http.ResponseWriter, r *http.Request) {
		serveLobbySocket(pool, w, r)
	})
	r.HandleFunc("/lobby", serveLobby)
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
