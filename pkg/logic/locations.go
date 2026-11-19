package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"os"
	"sort"
	"time"
)

type country struct {
	Name           string
	Size           float64 // original asset metadata, never a sampling weight
	SamplingWeight float64 `json:"-"`
	Areas          MultiPolygon
	StrictBounds   bool `json:"strictBounds"`
	bounds         [][]Ring
}

type countryStore struct {
	Countries map[string]*country
	totalSize float64
}

var countryDB countryStore
var CountryList []string

// Startup-only: validate all geometry/continent coverage before publishing the
// immutable catalog and sampling database. Reinitialization never accumulates
// weights or country codes; errors leave the previously valid data untouched.
func InitCountryDB() error {
	slog.Info("Populating countriesDB")
	b, err := os.ReadFile("assets/countryDB.json")
	if err != nil {
		return err
	}
	var loaded countryStore
	if err := json.Unmarshal(b, &loaded); err != nil {
		return err
	}
	if len(loaded.Countries) == 0 || len(loaded.Countries) > MaxCountryCodes {
		return ErrLocationData
	}
	codes := make([]string, 0, len(loaded.Countries))
	for code, country := range loaded.Countries {
		if len(code) != 2 || code[0] < 'A' || code[0] > 'Z' || code[1] < 'A' || code[1] > 'Z' ||
			country == nil || country.Name == "" || country.Size <= 0 || math.IsNaN(country.Size) || math.IsInf(country.Size, 0) {
			return ErrLocationData
		}
		buf, err := os.ReadFile(fmt.Sprintf("assets/basic/%s.json", code))
		if err != nil {
			return err
		}
		if err := json.Unmarshal(buf, &country.Areas); err != nil {
			return fmt.Errorf("invalid country areas %s: %w", code, err)
		}
		codes = append(codes, code)
	}
	sort.Slice(codes, func(i, j int) bool {
		a, b := loaded.Countries[codes[i]].Name, loaded.Countries[codes[j]].Name
		return a < b || (a == b && codes[i] < codes[j])
	})
	nextCatalog, byCode, err := buildCountryCatalog(loaded.Countries, codes)
	if err != nil {
		return err
	}
	nextPolygons, err := readAllPolygons()
	if err != nil {
		return err
	}
	for _, code := range codes {
		if len(nextPolygons[code]) == 0 {
			return fmt.Errorf("missing full country polygons %s: %w", code, ErrLocationData)
		}
		if country := loaded.Countries[code]; country.StrictBounds {
			if err := json.Unmarshal(nextPolygons[code], &country.bounds); err != nil || len(country.bounds) == 0 {
				return fmt.Errorf("invalid strict country bounds %s: %w", code, ErrLocationData)
			}
		}
	}
	// Preserve the exact existing weighted sampling policy, independently of
	// geographical center/span. Region configuration never writes these weights.
	for _, code := range codes {
		country := loaded.Countries[code]
		country.SamplingWeight = math.Pow(country.Size, 0.16)
		loaded.totalSize += country.SamplingWeight
		country.Areas.InnerSize = 0
		for _, polygon := range country.Areas.SearchArea {
			polygon.Size = int(math.Sqrt(float64(polygon.Size)))
			country.Areas.InnerSize += polygon.Size
		}
	}
	countryDB, CountryList, catalog, catalogByCode, PolyDB = loaded, codes, nextCatalog, byCode, nextPolygons
	return nil
}

const (
	locationBudget      = 6 * time.Second
	maxMetadataAttempts = 8
	maxPointSamples     = 1024
	retryDelay          = 100 * time.Millisecond
)

// RndLocation has one shared budget: at most 8 HTTP calls, 1024 candidate
// points and 6 seconds, including retries. No round state changes on failure.
func RndLocation(countryList []string, totalSize float64) (Coords, string, error) {
	return RndLocationContext(context.Background(), countryList, totalSize)
}

// RndLocationContext preserves the same budgets and permits lifecycle cancellation.
func RndLocationContext(parent context.Context, countryList []string, totalSize float64) (Coords, string, error) {
	if len(countryList) > 0 {
		var err error
		totalSize, err = SumCCListSize(countryList)
		if err != nil {
			return Coords{}, "", err
		}
	}
	ctx, cancel := context.WithTimeout(parent, locationBudget)
	defer cancel()
	var polygon Polygon
	var ccode string
	var lastErr error
	pointSamples := 0
	for attempt := 0; attempt < maxMetadataAttempts; attempt++ {
		if ctx.Err() != nil {
			return Coords{}, "", ErrLocationTimeout
		}
		if attempt%4 == 0 {
			var err error
			polygon, ccode, err = SelectRndArea(countryList, totalSize)
			if err != nil {
				return Coords{}, "", err
			}
		}
		bbox := polygon.Rings[0].Bound()
		var pt Point
		found := false
		for pointSamples < maxPointSamples {
			if ctx.Err() != nil {
				return Coords{}, "", ErrLocationTimeout
			}
			pointSamples++
			pt = RndPointWithinBox(bbox)
			if polygonContains(polygon.Rings, pt) {
				found = true
				break
			}
		}
		if !found {
			return Coords{}, "", ErrLocationData
		}
		loc, err := CheckStreetViewExists(ctx, pt, polygon.Radius)
		if ctx.Err() != nil {
			return Coords{}, "", ErrLocationTimeout
		}
		// A search radius can snap to imagery outside a curated area or country.
		// Opted-in countries reject that result within the same retry budget.
		if err == nil && !countryDB.Countries[ccode].allowsPanorama(polygon, loc) {
			err = ErrStreetViewNotFound
		}
		if err == nil {
			return loc, ccode, nil
		}
		lastErr = err
		if !errors.Is(err, ErrStreetViewNotFound) && !errors.Is(err, ErrProviderUnavailable) {
			return Coords{}, "", err
		}
		if errors.Is(err, ErrProviderUnavailable) && attempt+1 < maxMetadataAttempts {
			select {
			case <-ctx.Done():
				return Coords{}, "", ErrLocationTimeout
			case <-time.After(retryDelay):
			}
		}
	}
	return Coords{}, "", lastErr
}

// Legacy areas remain search hints. Strict countries make the selected polygon
// (including its holes) and the full country outline hard acceptance boundaries.
func (c *country) allowsPanorama(area Polygon, loc Coords) bool {
	if !c.StrictBounds {
		return true
	}
	point := Point{loc.Lng, loc.Lat}
	if !polygonContains(area.Rings, point) {
		return false
	}
	for _, rings := range c.bounds {
		if polygonContains(rings, point) {
			return true
		}
	}
	return false
}

// SelectRndArea fails safely if asset/configuration data is unavailable.
func SelectRndArea(countryList []string, totalSize float64) (Polygon, string, error) {
	ccode := SelectRandomCountry(countryList, totalSize)
	selected := countryDB.Countries[ccode]
	if selected == nil || selected.Areas.InnerSize <= 0 {
		return Polygon{}, "", ErrLocationData
	}
	rnd := rand.Intn(selected.Areas.InnerSize)
	for _, polygon := range selected.Areas.SearchArea {
		if polygon == nil || polygon.Size <= 0 {
			continue
		}
		if rnd < polygon.Size {
			if polygon.Radius <= 0 || len(polygon.Rings) == 0 {
				return Polygon{}, "", ErrLocationData
			}
			for _, ring := range polygon.Rings {
				if len(ring) < 3 {
					return Polygon{}, "", ErrLocationData
				}
				for _, pt := range ring {
					if !(Coords{Lat: pt[1], Lng: pt[0]}).Valid() {
						return Polygon{}, "", ErrLocationData
					}
				}
			}
			return *polygon, ccode, nil
		}
		rnd -= polygon.Size
	}
	return Polygon{}, "", ErrLocationData
}

// returns country code of a randomly selected country
func SelectRandomCountry(countryList []string, totalSize float64) string {
	if len(countryList) == 0 {
		if countryDB.totalSize <= 0 {
			return ""
		}
		rnd := rand.Float64() * countryDB.totalSize
		for ccode, country := range countryDB.Countries {
			if country == nil || country.SamplingWeight <= 0 {
				continue
			}
			if rnd <= country.SamplingWeight {
				return ccode
			}
			rnd -= country.SamplingWeight
		}
	} else {
		var err error
		totalSize, err = SumCCListSize(countryList)
		if err != nil || totalSize <= 0 {
			return ""
		}
		rnd := rand.Float64() * totalSize
		for _, ccode := range countryList {
			if rnd <= countryDB.Countries[ccode].SamplingWeight {
				return ccode
			}
			rnd -= countryDB.Countries[ccode].SamplingWeight
		}
	}
	return ""
}

// returns sum of selected countries size, rejecting unknown codes before use.
func SumCCListSize(countries []string) (float64, error) {
	var sizeSum float64
	for _, ccode := range countries {
		selected := countryDB.Countries[ccode]
		if selected == nil {
			return 0, errors.New("INVALID_COUNTRY_CODE")
		}
		if selected.SamplingWeight <= 0 || math.IsNaN(selected.SamplingWeight) || math.IsInf(selected.SamplingWeight, 0) {
			return 0, ErrLocationData
		}
		sizeSum += selected.SamplingWeight
	}
	return sizeSum, nil
}
