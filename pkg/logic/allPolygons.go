package logic

import (
	"encoding/json"
	"fmt"
	"os"
)

type property struct {
	Iso_a2 string `json:"ISO_A2_EH"`
}

type feature struct {
	Properties property
	Geometry   struct {
		Type        string
		Coordinates json.RawMessage
	}
}

var PolyDB map[string]json.RawMessage

// TODO: individually simplify country polygons; larger ones should be simplified more, smaller less
// Load a detached polygon map; InitCountryDB publishes it only after all assets validate.
func readAllPolygons() (map[string]json.RawMessage, error) {
	b, err := os.ReadFile("assets/full/countries2.json")
	if err != nil {
		return nil, fmt.Errorf("read full country polygons: %w", err)
	}
	var loaded struct {
		Type     string
		Features []feature
	}
	if err := json.Unmarshal(b, &loaded); err != nil {
		return nil, fmt.Errorf("decode full country polygons: %w", err)
	}
	if loaded.Type != "FeatureCollection" || len(loaded.Features) == 0 {
		return nil, ErrLocationData
	}
	partsByCode := make(map[string][]json.RawMessage)
	for _, feature := range loaded.Features {
		code := feature.Properties.Iso_a2
		if code == "" {
			return nil, ErrLocationData
		}
		var parts []json.RawMessage
		switch feature.Geometry.Type {
		case "Polygon":
			parts = []json.RawMessage{feature.Geometry.Coordinates}
		case "MultiPolygon":
			if err := json.Unmarshal(feature.Geometry.Coordinates, &parts); err != nil {
				return nil, fmt.Errorf("invalid full country polygons %s: %w", code, err)
			}
		default:
			return nil, fmt.Errorf("invalid full country geometry %s: %w", code, ErrLocationData)
		}
		if len(parts) == 0 {
			return nil, ErrLocationData
		}
		for _, part := range parts {
			var rings []Ring
			if err := json.Unmarshal(part, &rings); err != nil {
				return nil, fmt.Errorf("invalid full country rings %s: %w", code, err)
			}
			if len(rings) == 0 {
				return nil, ErrLocationData
			}
			for _, ring := range rings {
				if len(ring) < 4 || ring[0] != ring[len(ring)-1] {
					return nil, fmt.Errorf("invalid full country ring %s: %w", code, ErrLocationData)
				}
			}
		}
		// Keep every part for repeated codes without modifying decoded coordinate buffers.
		partsByCode[code] = append(partsByCode[code], parts...)
	}
	out := make(map[string]json.RawMessage, len(partsByCode))
	for code, parts := range partsByCode {
		coordinates, err := json.Marshal(parts)
		if err != nil {
			return nil, fmt.Errorf("encode full country polygons %s: %w", code, err)
		}
		out[code] = coordinates
	}
	return out, nil
}
