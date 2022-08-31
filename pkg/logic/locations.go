package logic

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
)

type country struct {
	Name string
	//CCode      string
	Size  float64
	Areas MultiPolygon
}

var countryDB struct {
	Countries map[string]*country
	totalSize float64
}

func InitCountryDB() {
	fmt.Println("Populating countriesDB")
	b, _ := os.ReadFile("assets/countryDB.json")
	if err := json.Unmarshal(b, &countryDB); err != nil {
		fmt.Println(err)
	}

	var sum float64
	// convert country sizen to 10th root and calculate size sum
	// populate every countries search area
	for ccode, country := range countryDB.Countries {
		country.Size = math.Pow(country.Size, 0.1)
		sum += country.Size
		buf, _ := os.ReadFile(fmt.Sprintf("assets/basic/%s.json", ccode))

		if err := json.Unmarshal(buf, &country.Areas); err != nil {
			fmt.Println(err)
		}

		for _, polygon := range country.Areas.SearchArea {
			country.Areas.InnerSize += polygon.Size
		}

		// fmt.Println(country.Areas)
	}
	countryDB.totalSize = sum

	// for i := 0; i < 10; i++ {
	// 	RndLocation()
	// 	//fmt.Println(ccode)
	// }
}

// returns valid random street view coordinates
func RndLocation() Coordinates {
	//fmt.Println(SelectRndArea())
	polygon := SelectRndArea()
	bbox := polygon.Rings[0].Bound()
	var status string
	var loc Coordinates
	var pt Point

	//fmt.Println("_START NEW LOCATION_")

	for apiOK, failCount := true, 0; apiOK; apiOK = (status == "ZERO_RESULTS") {
		// failsafe, if location repeatedly fails select different one
		if failCount >= 4 {
			fmt.Println("FAILSAFE ACTIVATED!")
			failCount = 0
			polygon = SelectRndArea()
			bbox = polygon.Rings[0].Bound()
		}

		for polyOK := true; polyOK; polyOK = !polygonContains(polygon.Rings, pt) {
			pt = RndPointWithinBox(bbox)
			fmt.Println("polygon contains: ", pt, polygonContains(polygon.Rings, pt))
		}
		loc, status = CheckStreetViewExists(pt, polygon.Radius)
		fmt.Println("api check: ", loc, status)
		failCount++
	}
	return loc

}

// returns random area name within random country
func SelectRndArea() Polygon {
	ccode := SelectRandomCountry()
	fmt.Println("Selected country: ", ccode)
	// for _, polygon := range countryDB.Countries[ccode].Areas.SearchArea {
	// 	fmt.Println(polygon)
	// }
	rnd := rand.Intn(countryDB.Countries[ccode].Areas.InnerSize)
	for area, polygon := range countryDB.Countries[ccode].Areas.SearchArea {
		if rnd <= polygon.Size {
			fmt.Println("Izbran polygon: ", area)
			return *polygon
		}
		rnd -= polygon.Size
	}
	return Polygon{}

}

// returns country code of a randomly selected country
func SelectRandomCountry() string {
	rnd := rand.Float64() * countryDB.totalSize
	for ccode, country := range countryDB.Countries {
		if rnd <= country.Size {
			return ccode
		}
		rnd -= country.Size
	}
	return ""
}
