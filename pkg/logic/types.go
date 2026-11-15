package logic

import (
	"encoding/json"
	"errors"
	"math"
)

type Coords struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

var ErrInvalidLocation = errors.New("INVALID_LOCATION")

func (c Coords) Valid() bool {
	return !math.IsNaN(c.Lat) && !math.IsInf(c.Lat, 0) && c.Lat >= -90 && c.Lat <= 90 &&
		!math.IsNaN(c.Lng) && !math.IsInf(c.Lng, 0) && c.Lng >= -180 && c.Lng <= 180
}

// Decode both components explicitly: absent/null is not a valid zero. Encoding
// always emits both axes, including explicitly supplied zero.
func (c *Coords) UnmarshalJSON(data []byte) error {
	var value struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return ErrInvalidLocation
	}
	if value.Lat == nil || value.Lng == nil {
		return ErrInvalidLocation
	}
	loc := Coords{Lat: *value.Lat, Lng: *value.Lng}
	if !loc.Valid() {
		return ErrInvalidLocation
	}
	*c = loc
	return nil
}
