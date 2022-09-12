package logic

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
)

type country struct {
	Name  string
	Size  float64
	Areas MultiPolygon
}

var countryDB struct {
	Countries map[string]*country
	totalSize float64
}
var CountryList []string

func InitCountryDB() {
	fmt.Println("Populating countriesDB")
	b, _ := os.ReadFile("assets/countryDB.json")
	if err := json.Unmarshal(b, &countryDB); err != nil {
		fmt.Println(err)
	}

	var sum float64
	// convert country size to 10th root and calculate size sum
	// populate every countries search area
	for ccode, country := range countryDB.Countries {
		country.Size = math.Pow(country.Size, 0.16)
		sum += country.Size
		buf, _ := os.ReadFile(fmt.Sprintf("assets/basic/%s.json", ccode))
		CountryList = append(CountryList, ccode)

		if err := json.Unmarshal(buf, &country.Areas); err != nil {
			fmt.Println(err)
		}

		for _, polygon := range country.Areas.SearchArea {
			polygon.Size = int(math.Sqrt(float64(polygon.Size)))
			country.Areas.InnerSize += polygon.Size
		}
	}
	sort.SliceStable(CountryList, func(i, j int) bool {
		return countryDB.Countries[CountryList[i]].Name < countryDB.Countries[CountryList[j]].Name
	})
	countryDB.totalSize = sum
}

// returns valid random street view coordinates
func RndLocation(countryList []string, totalSize float64) Coords {
	//fmt.Println(SelectRndArea())
	polygon := SelectRndArea(countryList, totalSize)
	bbox := polygon.Rings[0].Bound()
	var status string
	var loc Coords
	var pt Point

	for apiOK, failCount := true, 0; apiOK; apiOK = (status == "ZERO_RESULTS") {
		// failsafe, if location repeatedly fails select different one
		if failCount >= 4 {
			fmt.Println("FAILSAFE ACTIVATED!")
			failCount = 0
			polygon = SelectRndArea(countryList, totalSize)
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
func SelectRndArea(countryList []string, totalSize float64) Polygon {
	ccode := SelectRandomCountry(countryList, totalSize)
	fmt.Println("Selected country: ", ccode)
	rnd := rand.Intn(countryDB.Countries[ccode].Areas.InnerSize)
	for area, polygon := range countryDB.Countries[ccode].Areas.SearchArea {
		if rnd <= polygon.Size {
			fmt.Println("Selected polygon: ", area)
			return *polygon
		}
		rnd -= polygon.Size
	}
	return Polygon{}
}

// returns country code of a randomly selected country
func SelectRandomCountry(countryList []string, totalSize float64) string {
	if len(countryList) == 0 {
		rnd := rand.Float64() * countryDB.totalSize
		for ccode, country := range countryDB.Countries {
			if rnd <= country.Size {
				return ccode
			}
			rnd -= country.Size
		}
	} else {
		rnd := rand.Float64() * totalSize
		for _, ccode := range countryList {
			if rnd <= countryDB.Countries[ccode].Size {
				return ccode
			}
			rnd -= countryDB.Countries[ccode].Size
		}
	}
	return ""
}

// returns sum of selected countries size
func SumCCListSize(countries []string) float64 {
	var sizeSum float64
	for _, ccode := range countries {
		// TODO: fix crash when ccode doesnt exist
		sizeSum += countryDB.Countries[ccode].Size
	}
	return sizeSum
}
