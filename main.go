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

	"github.com/joho/godotenv"
)

// var Lobbies []lobby.Lobby
var Lobbies = []lobby.Lobby{
	{Name: "prvi lobby", ID: 45192, MaxPlayers: 8, NumPlayers: 0},
	{Name: "LOBBY #2", ID: 12334, MaxPlayers: 6, NumPlayers: 2},
}

func serveGetLobby(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(Lobbies)
}

func serveRndLocation(pool *websocket.Pool, w http.ResponseWriter, r *http.Request) {

	// Added query parameter reader for id of lobby
	id := r.URL.Query().Get("id")
	fmt.Println("logging ROOM id =>", id)

	fmt.Println("WebSocket Endpoint Hit")
	conn, err := websocket.Upgrade(w, r)
	if err != nil {
		fmt.Fprintf(w, "%+v\n", err)
	}

	client := &websocket.Client{
		Conn: conn,
		Pool: pool,
		Room: id,
	}

	pool.Register <- client

	client.Read()
}

func serveGetDistance(w http.ResponseWriter, r *http.Request) {
	// deal with CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	if r.Method == "OPTIONS" {
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	var guessLocation logic.Coordinates
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	if err = json.Unmarshal(reqBody, &guessLocation); err != nil {
		fmt.Println(err)
		return
	}

	var distance = logic.CalcDistance(logic.LastSentLoc, guessLocation)
	fmt.Println("Real loc: ", logic.LastSentLoc, ", Guess loc: ", guessLocation, ", Dist: ", distance)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct {
		Distance float64 `json:"distance"`
	}{distance})
}

func setupRoutes() {
	pool := websocket.NewPool()
	go pool.Start()
	http.HandleFunc("/getRndLocation", func(w http.ResponseWriter, r *http.Request) {
		serveRndLocation(pool, w, r)
	})
	http.HandleFunc("/getDistance", serveGetDistance)
	http.HandleFunc("/getLobby", serveGetLobby)
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
	setupRoutes()
	http.ListenAndServe("0.0.0.0:8080", nil)
}
