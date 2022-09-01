package logic

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

// calculates distance between 2 Coordinates using haversine formula
func CalcDistance(loc_1 Coordinates, loc_2 Coordinates) float64 {
	//fmt.Println("_REAL_LOC, USER LOC: ", loc_1, loc_2)
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
// func GenerateRndLocation() Coordinates {
// 	var status string
// 	var location Coordinates
// 	// alternative for do while, runs until a valid location is generated
// 	// TODO propperly handle all posible API responses
// 	for next := true; next; next = (status == "ZERO_RESULTS") {
// 		var lat = rand.Float64()*(48-44) + 44
// 		var lng = rand.Float64() * 7
// 		// fmt.Println("generated coordinates; ", lat, lng)
// 		location, status = validateLocation(lat, lng)
// 		// fmt.Println("api response: ", status, " pano location: ", location)
// 	}
// 	return location
// }

func RndPointWithinBox(b Bound) Point {
	lng := rand.Float64()*(b.Max[0]-b.Min[0]) + b.Min[0]
	lat := rand.Float64()*(b.Max[1]-b.Min[1]) + b.Min[1]
	return Point{lng, lat}
}

// checks if pano exists near requested location, returns exact location and status code
func CheckStreetViewExists(loc Point, radius int) (Coordinates, string) {
	res, err := http.Get(fmt.Sprintf("https://maps.googleapis.com/maps/api/streetview/metadata?location=%f,%f&key=%s&radius=%d&source=outdoor", loc[1], loc[0], os.Getenv("GMAPS_API"), radius))
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	var response MetadataResponse
	json.Unmarshal(body, &response)
	return response.Location, response.Status
}

// checks if pano exists near requested location, returns exact location and status code
// func validateLocation(lat float64, lng float64) (Coordinates, string) {
// 	res, err := http.Get(fmt.Sprintf("https://maps.googleapis.com/maps/api/streetview/metadata?location=%f,%f&key=%s&radius=500", lat, lng, os.Getenv("GMAPS_API")))
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	defer res.Body.Close()
// 	body, err := io.ReadAll(res.Body)
// 	if err != nil {
// 		fmt.Println(err)
// 	}

// 	var response MetadataResponse
// 	json.Unmarshal(body, &response)
// 	return response.Location, response.Status
// }

const letterBytes = "123456789ABCDEFGHJKLMNPRSTUVWXYZ"

var src = rand.NewSource(time.Now().UnixNano())

// Generates n long string (max 12) using 32 different characters from letterBytes
// src.Int63() generates 63 random bits, we use the last 5 as letterBytes index
// shift 5 places right & repeat; Simplified #7 from https://stackoverflow.com/a/31832326
// TODO verify unique ID
func GenerateRndID(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache := n-1, src.Int63(); i >= 0; {
		idx := int(cache & 31)
		sb.WriteByte(letterBytes[idx])
		i--
		cache >>= 5
	}

	return sb.String()
}
