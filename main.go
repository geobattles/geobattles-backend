package main

import (
	"fmt"
	"math"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type coordinates struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
}

var counter int = 0

// albums slice to seed record album data.
var locations = []coordinates{
	{Latitude: 42.345573, Longitude: -71.098326},
	{Latitude: 46.1080212, Longitude: 14.530384},
	{Latitude: -32.6967123, Longitude: 23.811269},
	{Latitude: 41.8875707, Longitude: 12.4944658},
}

func main() {
	router := gin.Default()
	router.GET("/getLocation", getLocation)
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

// getLocation cycles through available locations and responds with one.
func getLocation(c *gin.Context) {
	counter++
	if counter >= len(locations) {
		counter = 0
	}

	c.Header("Access-Control-Allow-Origin", "*")

	fmt.Println("Sent location: ", locations[counter], counter)
	c.IndentedJSON(http.StatusOK, locations[counter])
}

// getDistance reads guess coordinates and calculates distance to the right ones
// responds with distance in JSON
func getDistance(c *gin.Context) {
	var guessLocation coordinates

	if err := c.BindJSON(&guessLocation); err != nil {
		return
	}

	fmt.Println("correst location, guess: ", locations[counter], guessLocation)

	var distance float64 = calcDistance(locations[counter], guessLocation)
	fmt.Println("distance response: ", fmt.Sprintf(`{"distance": %f}`, distance))
	c.Header("Access-Control-Allow-Headers", "*")

	c.Header("Access-Control-Allow-Methods", "PUT, POST, GET, DELETE, OPTIONS")
	c.Header("Access-Control-Allow-Origin", "*")
	c.IndentedJSON(http.StatusOK, fmt.Sprintf(`{"distance": %f}`, distance))

}

// calculates distance between 2 sets of corrdinates using
// haversine formula https://en.wikipedia.org/wiki/Haversine_formula
func calcDistance(loc_1 coordinates, loc_2 coordinates) float64 {
	const R = 6371e3
	var fi_1 float64 = loc_1.Latitude * math.Pi / 180
	var fi_2 float64 = loc_2.Latitude * math.Pi / 180
	var delta_fi float64 = (loc_2.Latitude - loc_1.Latitude) * math.Pi / 180
	var delta_lambda float64 = (loc_2.Longitude - loc_1.Longitude) * math.Pi / 180

	var a float64 = math.Pow(math.Sin(delta_fi/2), 2) + math.Cos(fi_1)*math.Cos(fi_2)*math.Pow(math.Sin(delta_lambda/2), 2)
	var c = 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
