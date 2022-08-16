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
	fmt.Println(id)
	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(lobby.LobbyList)
		fmt.Println("Sent lobby list")
	case http.MethodPost:
		var newLobby lobby.Lobby
		newLobby.Results = make(map[int]map[string][]float64)
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		if err = json.Unmarshal(reqBody, &newLobby); err != nil {
			fmt.Println(err)
			return
		}
		newLobby.ID = logic.GenerateRndID(6)
		lobby.LobbyList = append(lobby.LobbyList, newLobby)
		fmt.Println("Created lobby ", newLobby.ID)
		json.NewEncoder(w).Encode(lobby.LobbyList)
	case http.MethodDelete:
		for index, value := range lobby.LobbyList {
			if value.ID == id {
				lobby.LobbyList = append(lobby.LobbyList[:index], lobby.LobbyList[index+1:]...)
				fmt.Println("Deleted lobby ", id)
			}
		}
	}
}

func serveLobbySocket(pool *websocket.Pool, w http.ResponseWriter, r *http.Request) {
	// Added query parameter reader for id of lobby
	lobbyID := r.URL.Query().Get("id")
	userName := r.URL.Query().Get("name")
	fmt.Println("WebSocket Endpoint Hit, room ID: ", lobbyID, " name: ", userName)
	// only connect to ws if lobby exists
	if func() bool {
		for _, value := range lobby.LobbyList {
			if value.ID == lobbyID {
				return true
			}
		}
		return false
	}() {
		fmt.Println("correct id, upgrading ws")
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
		lobby.AddPlayerToLobby(lobby.LobbyList, client.Name, lobbyID)

		pool.Register <- client
		fmt.Println("client: ", client)
		fmt.Println("lobbyList: ", lobby.LobbyList)

		client.Read()
		fmt.Println("po client.read()")
	}
}

// func serveGetDistance(w http.ResponseWriter, r *http.Request) {
// 	// deal with CORS
// 	w.Header().Set("Access-Control-Allow-Origin", "*")
// 	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
// 	w.Header().Set("Access-Control-Allow-Headers", "*")
// 	if r.Method == "OPTIONS" {
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json; charset=utf-8")
// 	var guessLocation logic.Coordinates
// 	reqBody, err := io.ReadAll(r.Body)
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}
// 	if err = json.Unmarshal(reqBody, &guessLocation); err != nil {
// 		fmt.Println(err)
// 		return
// 	}
// var distance = logic.CalcDistance(logic.LastSentLoc, guessLocation)
// fmt.Println("Real loc: ", logic.LastSentLoc, ", Guess loc: ", guessLocation, ", Dist: ", distance)
// w.WriteHeader(http.StatusOK)
// json.NewEncoder(w).Encode(struct {
// 	Distance float64 `json:"distance"`
// }{distance})
// }

func setupRoutes(r *mux.Router) {
	pool := websocket.NewPool()
	go pool.Start()
	r.HandleFunc("/lobbySocket", func(w http.ResponseWriter, r *http.Request) {
		serveLobbySocket(pool, w, r)
	})
	//r.HandleFunc("/getDistance", serveGetDistance)
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
