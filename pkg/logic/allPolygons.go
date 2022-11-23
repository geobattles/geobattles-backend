package logic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

type property struct {
	Iso_a2 string `json:"ISO_A2_EH"`
}

type feature struct {
	Properties property
	Geometry   map[string]json.RawMessage
}

var importAll struct {
	Features []feature
}

var PolyDB map[string]json.RawMessage

// TODO: individually simplify country polygons; larger ones should be simplified more, smaller less
// helper function to populate PolyDB from geojson on disk
func readAllPolygons() {
	PolyDB = make(map[string]json.RawMessage)

	b, _ := os.ReadFile("assets/full/countries2.json")
	if err := json.Unmarshal(b, &importAll); err != nil {
		fmt.Println(err)
	}
	// if feature (country) has a single polygon convert it to a multipolygon for simplicity
	for _, feature := range importAll.Features {
		if bytes.Equal([]byte(`"Polygon"`), feature.Geometry["type"]) {
			if PolyDB[feature.Properties.Iso_a2] == nil {
				PolyDB[feature.Properties.Iso_a2] = append([]byte(`[`), append(feature.Geometry["coordinates"], []byte(`]`)...)...)
			} else {
				PolyDB[feature.Properties.Iso_a2] = PolyDB[feature.Properties.Iso_a2][:len(PolyDB[feature.Properties.Iso_a2])-1]
				PolyDB[feature.Properties.Iso_a2] = append(PolyDB[feature.Properties.Iso_a2], append([]byte(`,`), append(feature.Geometry["coordinates"], []byte(`]`)...)...)...)

			}

		} else {
			if PolyDB[feature.Properties.Iso_a2] == nil {
				PolyDB[feature.Properties.Iso_a2] = feature.Geometry["coordinates"]
			} else {
				PolyDB[feature.Properties.Iso_a2] = PolyDB[feature.Properties.Iso_a2][:len(PolyDB[feature.Properties.Iso_a2])-1]
				feature.Geometry["coordinates"] = append(feature.Geometry["coordinates"][:0], feature.Geometry["coordinates"][1:]...)
				PolyDB[feature.Properties.Iso_a2] = append(PolyDB[feature.Properties.Iso_a2], append([]byte(`,`), feature.Geometry["coordinates"]...)...)
			}
		}
	}
}
