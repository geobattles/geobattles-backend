package reverse

import (
	"encoding/json"
	"log/slog"
	"os"
)

type point [2]float64
type ring []point
type polygon []ring
type multiPolygon []polygon

type geometry struct {
	TypeGeometry string       `json:"type"`
	Coordinates  multiPolygon `json:"coordinates"`
}

type feature struct {
	TypeFeature string            `json:"type"`
	Properties  map[string]string `json:"properties"`
	Bbox        [4]float64        `json:"bbox"`
	Geometry    geometry          `json:"geometry"`
}

type geojson struct {
	Type     string    `json:"type"`
	Features []feature `json:"features"`
}

var fullPolygons geojson

// read fullPolygons geojson and store it in local struct
func InitReverse() error {
	slog.Info("Reading reverse.json")
	b, _ := os.ReadFile("assets/reverse.json")
	if err := json.Unmarshal(b, &fullPolygons); err != nil {
		slog.Error("Error unmarshalling reverse.json", "error", err.Error())
		return err
	}

	return nil
}
