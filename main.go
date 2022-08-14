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

//var LobbyList []lobby.Lobby

var LobbyList = []lobby.Lobby{
	{Name: "prvi lobby", ID: "U4YPR6", MaxPlayers: 8, NumPlayers: 0},
	{Name: "LOBBY #2", ID: "8CKXRG", MaxPlayers: 6, NumPlayers: 2},
}

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
	fmt.Println(id)
	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(LobbyList)
		fmt.Println("Sent lobby list")
	case http.MethodPost:
		var lobby lobby.Lobby
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		if err = json.Unmarshal(reqBody, &lobby); err != nil {
			fmt.Println(err)
			return
		}
		lobby.ID = logic.GenerateRndID(6)
		LobbyList = append(LobbyList, lobby)
		fmt.Println("Created lobby ", lobby.ID)
		json.NewEncoder(w).Encode(LobbyList)
	case http.MethodDelete:
		for index, lobby := range LobbyList {
			if lobby.ID == id {
				LobbyList = append(LobbyList[:index], LobbyList[index+1:]...)
				fmt.Println("Deleted lobby ", id)
			}
		}
	}

}

func serveRndLocation(pool *websocket.Pool, w http.ResponseWriter, r *http.Request) {

	// Added query parameter reader for id of lobby
	id := r.URL.Query().Get("id")
	fmt.Println("WebSocket Endpoint Hit, room ID: ", id)
	// only connect to ws if lobby exists
	if func() bool {
		for _, value := range LobbyList {
			if value.ID == id {
				return true
			}
		}
		return false
	}() {
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

func setupRoutes(r *mux.Router) {
	pool := websocket.NewPool()
	go pool.Start()
	r.HandleFunc("/getRndLocation", func(w http.ResponseWriter, r *http.Request) {
		serveRndLocation(pool, w, r)
	})
	r.HandleFunc("/getDistance", serveGetDistance)
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
	http.ListenAndServe("0.0.0.0:8080", router)
}
