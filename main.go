package main

import (
	"encoding/json"
	"example/web-service-gin/pkg/logic"
	"example/web-service-gin/pkg/websocket"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/joho/godotenv"
)

func serveRndLocation(pool *websocket.Pool, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "PUT, POST, GET, DELETE, OPTIONS")
	fmt.Println("WebSocket Endpoint Hit")
	conn, err := websocket.Upgrade(w, r)
	if err != nil {
		fmt.Fprintf(w, "%+v\n", err)
	}

	client := &websocket.Client{
		Conn: conn,
		Pool: pool,
	}

	pool.Register <- client
	client.Read()
}

func serveGetDistance(w http.ResponseWriter, r *http.Request) {
	var guessLocation logic.Coordinates
	reqBody, err := ioutil.ReadAll(r.Body)
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

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "PUT, POST, GET, DELETE, OPTIONS")
	w.Header().Set("Content-Type", "application/json")
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

// func main() {
// 	//set seed for rand function
// 	rand.Seed(time.Now().UnixNano())
// 	// try to read .env file
// 	// in docker we just use ENV variables and this WILL throw error
// 	err := godotenv.Load()
// 	if err != nil {
// 		log.Println("Error loading .env file")
// 	}

// 	router := gin.Default()
// 	go pool.Start()
// 	router.GET("/getRndLocation", getRndLocation)
// 	router.POST("/getDistance", getDistance)

// 	router.Use(cors.New(cors.Config{
// 		AllowOrigins: []string{"*"},
// 		AllowMethods: []string{"GET", "POST", "OPTIONS"},
// 		AllowHeaders: []string{"*"},
// 		//ExposeHeaders:    []string{"Content-Length"},
// 		//MaxAge: 12 * time.Hour,
// 	}))
// 	router.Run("0.0.0.0:8080")
// }

// // sends random valid coordinates
// func getRndLocation(c *gin.Context) {
// 	fmt.Println("WebSocket Endpoint Hit")
// 	c.Header("Access-Control-Allow-Origin", "*")

// 	conn, err := websocket.Upgrade(c.Writer, c.Request)
// 	if err != nil {
// 		fmt.Fprintf(c.Writer, "%+v\n", err)
// 	}

// 	client := &websocket.Client{
// 		Conn: conn,
// 		Pool: pool,
// 	}

// 	pool.Register <- client
// 	client.Read()
// }

// getDistance reads guess coordinates and calculates distance to the right ones
// responds with distance in JSON
// func getDistance(c *gin.Context) {
// 	var guessLocation logic.Coordinates

// 	if err := c.BindJSON(&guessLocation); err != nil {
// 		return
// 	}

// 	fmt.Println("correst location; guess: ", logic.LastSentLoc, guessLocation)

// 	var distance float64 = logic.CalcDistance(logic.LastSentLoc, guessLocation)
// 	fmt.Println("distance response: ", fmt.Sprintf(`{"distance": %f}`, distance))
// 	c.Header("Access-Control-Allow-Headers", "*")

// 	c.Header("Access-Control-Allow-Methods", "PUT, POST, GET, DELETE, OPTIONS")
// 	c.Header("Access-Control-Allow-Origin", "*")
// 	c.IndentedJSON(http.StatusOK, fmt.Sprintf(`{"distance": %f}`, distance))
// }
