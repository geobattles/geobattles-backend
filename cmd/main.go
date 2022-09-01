// Helper package for testing country polygons and api
// run with: go run cmd\main.go <poly or api> <CCODE> <poly name, optional>
// ex: "cmd\main.go poly AU Australia1" tests polygon responses on Australia1 polygon
// ex: "cmd\main.go api AU" tests api responses for all polygons in AU
// the idea is to first test polygons (default 100 times each) to see if they are optimal
// this is cheap so even high values like 3+ shouldnt be a problem
// next is testing actual api for street view availability (10 times for each polygon)
// search radius should be tuned to reach value of around ~1.2
// lower than that means results arent accurate and higher is wasting api calls
// *while testing there is a limit of 1 api req/s, it can take a few mins for more complex countries
package main

import (
	"encoding/json"
	"example/web-service-gin/pkg/logic"
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type country struct {
	Name string
	//CCode      string
	Size  float64
	Areas logic.MultiPolygon
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
	}
	countryDB.totalSize = sum
}

// checks if a point is inside Polygon
func polygonContains(p []logic.Ring, pt logic.Point) bool {
	// if point is not within the outer ring return false
	if !ringContains(p[0], pt) {
		return false
	}
	// if point is within a hole return false
	for i := 1; i < len(p); i++ {
		if ringContains(p[i], pt) {
			return false
		}
	}
	return true
}

// checks if ring contains a point
func ringContains(r logic.Ring, pt logic.Point) bool {
	// if point is not within ring bounds return false
	if !r.Bound().Contains(pt) {
		return false
	}

	c, on := rayIntersect(pt, r[0], r[len(r)-1])
	if on {
		return true
	}

	for i := 0; i < len(r)-1; i++ {
		inter, on := rayIntersect(pt, r[i], r[i+1])
		if on {
			return true
		}

		if inter {
			c = !c
		}
	}

	return c
}

// checks if point intersects a segment or lies on it
func rayIntersect(p, s, e logic.Point) (intersects, on bool) {
	if s[0] > e[0] {
		s, e = e, s
	}

	if p[0] == s[0] {
		if p[1] == s[1] {
			// p == start
			return false, true
		} else if s[0] == e[0] {
			// vertical segment (s -> e)
			// return true if within the line, check to see if start or end is greater.
			if s[1] > e[1] && s[1] >= p[1] && p[1] >= e[1] {
				return false, true
			}

			if e[1] > s[1] && e[1] >= p[1] && p[1] >= s[1] {
				return false, true
			}
		}

		// Move the y coordinate to deal with degenerate case
		p[0] = math.Nextafter(p[0], math.Inf(1))
	} else if p[0] == e[0] {
		if p[1] == e[1] {
			// matching the end point
			return false, true
		}

		p[0] = math.Nextafter(p[0], math.Inf(1))
	}

	if p[0] < s[0] || p[0] > e[0] {
		return false, false
	}

	if s[1] > e[1] {
		if p[1] > s[1] {
			return false, false
		} else if p[1] < e[1] {
			return true, false
		}
	} else {
		if p[1] > e[1] {
			return false, false
		} else if p[1] < s[1] {
			return true, false
		}
	}

	rs := (p[1] - s[1]) / (p[0] - s[0])
	ds := (e[1] - s[1]) / (e[0] - s[0])

	if rs == ds {
		return false, true
	}

	return rs <= ds, false
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("not enough args")
		return
	}
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}
	rand.Seed(time.Now().UnixNano())

	mode := os.Args[1]
	ccode := os.Args[2]
	var polyName string
	if len(os.Args) > 3 {
		polyName = os.Args[3]
	}
	InitCountryDB()
	fmt.Println("CHOSEN MODE: ", mode)
	fmt.Println("CHOSEN CCODE: ", ccode)

	var totalResults string
	var polygonList []string
	if polyName != "" {
		polygonList = append(polygonList, polyName)
	} else {
		for area := range countryDB.Countries[ccode].Areas.SearchArea {
			polygonList = append(polygonList, area)
		}
	}
	fmt.Println("number of polygons selected: ", len(polygonList))
	fmt.Println(polygonList)
	fmt.Println()

	switch mode {
	case "poly":
		tries := 100
		for i := 0; i < len(polygonList); i++ {
			polygon := countryDB.Countries[ccode].Areas.SearchArea[polygonList[i]]
			//fmt.Println("CURRENT POLYGON: ", polygonList[i])
			bbox := polygon.Rings[0].Bound()
			var pt logic.Point
			var polyAttempt int
			for j := 0; j < tries; j++ {
				for polyOK := true; polyOK; polyOK = !polygonContains(polygon.Rings, pt) {
					pt = logic.RndPointWithinBox(bbox)
					fmt.Println("poly contains: ", pt, polygonContains(polygon.Rings, pt))
					polyAttempt++
				}

			}
			fmt.Println()
			fmt.Println("CURRENT POLYGON: ", polygonList[i])
			fmt.Println("POLY_ATTEMPT:    ", polyAttempt, float64(polyAttempt)/float64(tries))
			totalResults += fmt.Sprintln("POLYGON:      ", polygonList[i])
			totalResults += fmt.Sprintln("POLY_ATTEMPT: ", polyAttempt, float64(polyAttempt)/float64(tries))

		}
	case "api":
		tries := 10
		for i := 0; i < len(polygonList); i++ {
			polygon := countryDB.Countries[ccode].Areas.SearchArea[polygonList[i]]
			bbox := polygon.Rings[0].Bound()
			var status string
			var loc logic.Coords
			var pt logic.Point
			var apiFail, apiAttempt, polyAttempt int
			for j := 0; j < tries; j++ {
				for apiOK, failsafe := true, 0; apiOK; apiOK = (status == "ZERO_RESULTS") {
					// failsafe, if location repeatedly fails select different one
					if failsafe >= 7 {
						fmt.Println("FAILSAFE ACTIVATED!")
						apiFail++
						break
					}

					for polyOK := true; polyOK; polyOK = !polygonContains(polygon.Rings, pt) {
						pt = logic.RndPointWithinBox(bbox)
						polyAttempt++
					}
					time.Sleep(1000 * time.Millisecond)
					loc, status = logic.CheckStreetViewExists(pt, polygon.Radius)
					fmt.Println("api check: ", pt, loc, status)
					failsafe++
					apiAttempt++
				}

			}
			fmt.Println()
			fmt.Println("CURRENT POLYGON: ", polygonList[i])
			fmt.Println("API_ATTEMPTS:    ", apiAttempt, float64(apiAttempt)/float64(tries))
			fmt.Println("API_FAILS:       ", apiFail, float64(apiFail)/float64(tries))
			fmt.Println("POLY_ATTEMPT:    ", polyAttempt, float64(apiFail)/float64(tries))
			totalResults += fmt.Sprintln("POLYGON:      ", polygonList[i])
			totalResults += fmt.Sprintln("POLY_ATTEMPT: ", polyAttempt, float64(polyAttempt)/float64(tries))
			totalResults += fmt.Sprintln("API_ATTEMPTS: ", apiAttempt, float64(apiAttempt)/float64(tries))
			totalResults += fmt.Sprintln("API_FAILS:    ", apiFail, float64(apiFail)/float64(tries))

		}
	}
	fmt.Println("\nTOTAL RESULTS\n", totalResults)

}
