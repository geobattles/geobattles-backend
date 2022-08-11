package main

import (
	"example/web-service-gin/pkg/logic"
	"example/web-service-gin/pkg/websocket"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var pool = websocket.NewPool()

func main() {
	//set seed for rand function
	rand.Seed(time.Now().UnixNano())
	// try to read .env file
	// in docker we just use ENV variables and this WILL throw error
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	router := gin.Default()
	go pool.Start()
	router.GET("/getRndLocation", getRndLocation)
	router.POST("/getDistance", getDistance)

	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "OPTIONS"},
		AllowHeaders: []string{"*"},
		//ExposeHeaders:    []string{"Content-Length"},
		//MaxAge: 12 * time.Hour,
	}))
	router.Run("0.0.0.0:8080")
}

// sends random valid coordinates
func getRndLocation(c *gin.Context) {
	fmt.Println("WebSocket Endpoint Hit")
	c.Header("Access-Control-Allow-Origin", "*")

	conn, err := websocket.Upgrade(c.Writer, c.Request)
	if err != nil {
		fmt.Fprintf(c.Writer, "%+v\n", err)
	}

	client := &websocket.Client{
		Conn: conn,
		Pool: pool,
	}

	pool.Register <- client
	client.Read()
}

// getDistance reads guess coordinates and calculates distance to the right ones
// responds with distance in JSON
func getDistance(c *gin.Context) {
	var guessLocation logic.Coordinates

	if err := c.BindJSON(&guessLocation); err != nil {
		return
	}

	fmt.Println("correst location; guess: ", logic.LastSentLoc, guessLocation)

	var distance float64 = logic.CalcDistance(logic.LastSentLoc, guessLocation)
	fmt.Println("distance response: ", fmt.Sprintf(`{"distance": %f}`, distance))
	c.Header("Access-Control-Allow-Headers", "*")

	c.Header("Access-Control-Allow-Methods", "PUT, POST, GET, DELETE, OPTIONS")
	c.Header("Access-Control-Allow-Origin", "*")
	c.IndentedJSON(http.StatusOK, fmt.Sprintf(`{"distance": %f}`, distance))
}
