package logic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
)

func CalcDistance(loc_1 Coordinates, loc_2 Coordinates) float64 {
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
func GenerateRndLocation() Coordinates {
	var status string
	var location Coordinates
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
func validateLocation(lat float64, lng float64) (Coordinates, string) {
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
