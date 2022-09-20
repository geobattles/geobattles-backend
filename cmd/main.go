// Helper package for testing country polygons and api
// run with: go run cmd\main.go <poly or api or rnd> <CCODE> <poly name, optional>
// ex: "cmd\main.go poly XX xxxx" tests polygon responses on xxxx polygon
// ex: "cmd\main.go api XX" tests api responses for all polygons in XX
// ex: "cmd\main.go rnd" tests random pick rate of countries
// ex: "cmd\main.go rnd XX" tests random pick rate of areas within XX

// the idea is to first test polygons to see if they are optimal
// this is cheap so even high values like 3+ shouldnt be a problem
// next is testing actual api for street view availability (25 times for each polygon)
// search radius should be tuned to reach value of around ~1.2
// lower than that means results arent accurate and higher is wasting api calls
// *while testing there is a limit of 5 api req/s, it can take a few mins for more complex countries
// rnd gives an idea of how likely a country/area is to be selected

package main

import (
	"encoding/json"
	"example/web-service-gin/pkg/logic"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"text/tabwriter"
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
	if len(os.Args) < 2 {
		fmt.Println("Not enough args")
		return
	}
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}
	rand.Seed(time.Now().UnixNano())

	mode := os.Args[1]
	if mode != "rnd" && len(os.Args) < 3 {
		fmt.Println("Not enough args")
		return
	}
	InitCountryDB()
	fmt.Println("CHOSEN MODE: ", mode)

	var polyName string
	if len(os.Args) > 3 {
		polyName = os.Args[3]
	}
	var ccode string
	var polygonList []string
	if len(os.Args) > 2 {
		ccode = os.Args[2]
		fmt.Println("CHOSEN CCODE: ", ccode)
		if polyName != "" {
			polygonList = append(polygonList, polyName)
		} else {
			for area := range countryDB.Countries[ccode].Areas.SearchArea {
				polygonList = append(polygonList, area)
			}
		}
		sort.Strings(polygonList)
		fmt.Println("number of polygons selected: ", len(polygonList))
		fmt.Println(polygonList)
	}

	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)

	switch mode {
	case "poly":
		tries := 1000
		fmt.Fprintln(w, "AREA\t POLY %")

		for i := 0; i < len(polygonList); i++ {
			polygon := countryDB.Countries[ccode].Areas.SearchArea[polygonList[i]]
			//fmt.Println("CURRENT POLYGON: ", polygonList[i])
			bbox := polygon.Rings[0].Bound()
			var pt logic.Point
			var polyAttempt int
			for j := 0; j < tries; j++ {
				for polyOK := true; polyOK; polyOK = !polygonContains(polygon.Rings, pt) {
					pt = logic.RndPointWithinBox(bbox)
					// fmt.Println("poly contains: ", pt, polygonContains(polygon.Rings, pt))
					polyAttempt++
				}

			}
			fmt.Fprintln(w, polygonList[i], "\t", fmt.Sprintf("%.2f", float64(polyAttempt)/float64(tries)))
		}
	case "api":
		tries := 25
		fmt.Fprintln(w, "AREA\t POLY %\t FAIL %\t API %")

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
					if failsafe >= 3 {
						fmt.Println("FAILSAFE ACTIVATED!")
						apiFail++
						break
					}

					for polyOK := true; polyOK; polyOK = !polygonContains(polygon.Rings, pt) {
						pt = logic.RndPointWithinBox(bbox)
						polyAttempt++
					}
					time.Sleep(200 * time.Millisecond)
					loc, status = logic.CheckStreetViewExists(pt, polygon.Radius)
					fmt.Println("api check: ", pt, loc, status)
					failsafe++
					apiAttempt++
				}
			}

			fmt.Fprintln(w, polygonList[i], "\t", fmt.Sprintf("%.2f", float64(polyAttempt)/float64(tries)), "\t", fmt.Sprintf("%.2f", float64(apiFail)/float64(tries)), "\t", fmt.Sprintf("%.2f", float64(apiAttempt)/float64(tries)))
		}
	case "rnd":
		tries := 1000000
		cLog := make(map[string]int)
		if len(polygonList) == 0 {
			for i := 0; i < tries; i++ {
				rnd := rand.Float64() * countryDB.totalSize
				for ccode, country := range countryDB.Countries {
					if rnd <= country.Size {
						cLog[ccode]++
						break
					}
					rnd -= country.Size
				}
			}
			fmt.Fprintln(w, "COUNTRY\t COUNT\t %")
			for _, ccode := range CountryList {
				fmt.Fprintln(w, countryDB.Countries[ccode].Name, "\t", cLog[ccode], "\t", fmt.Sprintf("%.2f", (float64(cLog[ccode])/float64(tries))*100))
			}
		} else {
			for i := 0; i < tries; i++ {

				rnd := rand.Intn(countryDB.Countries[ccode].Areas.InnerSize)
				for area, polygon := range countryDB.Countries[ccode].Areas.SearchArea {
					if rnd <= polygon.Size {
						cLog[area]++
						break
					}
					rnd -= polygon.Size
				}
			}
			fmt.Fprintln(w, "AREA\t COUNT\t %")
			for _, poly := range polygonList {
				fmt.Fprintln(w, poly, "\t", cLog[poly], "\t", fmt.Sprintf("%.2f", (float64(cLog[poly])/float64(tries))*100))
			}
		}

	}
	fmt.Println("\nRESULTS")
	w.Flush()

}
