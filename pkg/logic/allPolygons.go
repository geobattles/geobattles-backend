package logic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

type property struct {
	Iso_a2 string `json:"ISO_A2"`
}

type geometry struct {
	Polygon map[string]json.RawMessage
}

type feature struct {
	Properties property
	Geometry   map[string]json.RawMessage
}

type AllPolygons struct {
	Poly json.RawMessage
	// CC   string
}

var importAll struct {
	Features []feature
}

var PolyDB map[string]json.RawMessage

func readAllPolygons() {
	PolyDB = make(map[string]json.RawMessage)
	fmt.Println("reading all polygons")

	b, _ := os.ReadFile("assets/full/countries.json")
	if err := json.Unmarshal(b, &importAll); err != nil {
		fmt.Println(err)
	}
	// fmt.Println(importAll.Features[0].Geometry["type"])
	// fmt.Printf("%T", importAll.Features[0].Geometry["type"])
	// fmt.Println([]byte(`"Polygon"`))
	// fmt.Println(bytes.Compare([]byte(`"Polygon"`), importAll.Features[0].Geometry["type"]))
	// fmt.Println(append([]byte(`[`), append(importAll.Features[0].Geometry["coordinates"], []byte(`[`)...)...))

	for _, feature := range importAll.Features {
		if bytes.Equal([]byte(`"Polygon"`), feature.Geometry["type"]) {
			PolyDB[feature.Properties.Iso_a2] = append([]byte(`[`), append(feature.Geometry["coordinates"], []byte(`]`)...)...)
		} else {
			PolyDB[feature.Properties.Iso_a2] = feature.Geometry["coordinates"]
		}

	}

}
