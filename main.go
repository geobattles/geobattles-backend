package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type coordinates struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
}

type MetadataResponse struct {
	Location coordinates `json:"location"`
	Status   string
}

var counter int = 0

// albums slice to seed record album data.
var locations = []coordinates{
	{Latitude: 42.345573, Longitude: -71.098326},
	{Latitude: 46.1080212, Longitude: 14.530384},
	{Latitude: -32.6967123, Longitude: 23.811269},
	{Latitude: 41.8875707, Longitude: 12.4944658},
	{Latitude: 46.0856361, Longitude: 14.4226494},
	{Latitude: 38.9845741, Longitude: -3.9266799},
}

func main() {
	//set seed for rand function
	rand.Seed(time.Now().UnixNano())
	//try to read .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	router := gin.Default()
	router.GET("/getLocation", getLocation)
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

// sends random valid coordinates
func getRndLocation(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	loc := generateRndLocation()
	fmt.Println("Sent location: ", loc)
	c.IndentedJSON(http.StatusOK, loc)
}

// getDistance reads guess coordinates and calculates distance to the right ones
// responds with distance in JSON
func getDistance(c *gin.Context) {
	var guessLocation coordinates

	if err := c.BindJSON(&guessLocation); err != nil {
		return
	}

	fmt.Println("correst location; guess: ", locations[counter], guessLocation)

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

// generates random coordinates
func generateRndLocation() coordinates {
	var status string
	var location coordinates
	// alternative for do while, runs until a valid location is generated
	for next := true; next; next = (status != "OK") {
		var lat = rand.Float64()*(48-44) + 44
		var lng = rand.Float64() * 7
		fmt.Println("generated coordinates; ", lat, lng)
		location, status = validateLocation(lat, lng)
		fmt.Println("api response: ", status, " pano location: ", location)
	}
	return location
}

// checks if pano exists near requested location, returns exact location and status code
func validateLocation(lat float64, lng float64) (coordinates, string) {
	res, err := http.Get(fmt.Sprintf("https://maps.googleapis.com/maps/api/streetview/metadata?location=%f,%f&key=%s&radius=500", lat, lng, os.Getenv("GMAPS_API")))
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	var response MetadataResponse
	json.Unmarshal(body, &response)
	return response.Location, response.Status
}
