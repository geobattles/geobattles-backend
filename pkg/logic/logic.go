package logic

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

// calculates distance between 2 Coordinates using haversine formula
func CalcDistance(loc_1 Coords, loc_2 Coords) float64 {
	const R = 6371e3
	var fi_1 float64 = loc_1.Lat * math.Pi / 180
	var fi_2 float64 = loc_2.Lat * math.Pi / 180
	var delta_fi float64 = (loc_2.Lat - loc_1.Lat) * math.Pi / 180
	var delta_lambda float64 = (loc_2.Lng - loc_1.Lng) * math.Pi / 180

	var a float64 = math.Pow(math.Sin(delta_fi/2), 2) + math.Cos(fi_1)*math.Cos(fi_2)*math.Pow(math.Sin(delta_lambda/2), 2)
	var c = 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func RndPointWithinBox(b Bound) Point {
	lng := rand.Float64()*(b.Max[0]-b.Min[0]) + b.Min[0]
	lat := rand.Float64()*(b.Max[1]-b.Min[1]) + b.Min[1]
	return Point{lng, lat}
}

// checks if pano exists near requested location, returns exact location and status code
func CheckStreetViewExists(loc Point, radius int) (Coords, string) {
	res, err := http.Get(fmt.Sprintf("https://maps.googleapis.com/maps/api/streetview/metadata?location=%f,%f&key=%s&radius=%d&source=outdoor", loc[1], loc[0], os.Getenv("GMAPS_API"), radius))
	if err != nil {
		slog.Error("Error checking streetview metadata", "error", err.Error())
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("Error reading streetview metadata", "error", err.Error())
	}

	var response ApiMetaResponse
	json.Unmarshal(body, &response)
	return response.Loc, response.Status
}

const letterBytes = "123456789ABCDEFGHJKLMNPRSTUVWXYZ"

var src = rand.NewSource(time.Now().UnixNano())

// TODO: move to utils
// Generates n long string (max 12) using 32 different characters from letterBytes
// src.Int63() generates 63 random bits, we use the last 5 as letterBytes index
// shift 5 places right & repeat; Simplified #7 from https://stackoverflow.com/a/31832326
func GenerateRndID(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	// A src.Int63() generates 63 random bits, enough for 12 x 5bits!
	for i, cache := n-1, src.Int63(); i >= 0; {
		idx := int(cache & 31)
		sb.WriteByte(letterBytes[idx])
		i--
		cache >>= 5
	}

	return sb.String()
}
