package logic

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
)

const (
	EarthRadiusKm = 6371.0
	WorldSpanKm   = math.Pi * EarthRadiusKm
	MinimumHalfKm = 1.0
	PerfectMeters = 100.0
	// Fixed safety bound, not the catalog count; allow new countries without
	// changing validation each time. HTTP/socket byte limits still apply.
	MaxCountryCodes = 256
)

// LegacyHalfKm is the exponential half-score decay distance, not the perfect
// radius. Manual mode retains its original factor-dependent perfect radius.
func LegacyHalfKm(factor int) float64 {
	return math.Log(0.5) / math.Log(0.999) / (0.2 + 30*math.Pow(0.98, 10*math.Pow(float64(factor), 0.6)))
}

type CountryInfo struct {
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Continent string  `json:"continent"`
	Center    Coords  `json:"center"`
	SpanKm    float64 `json:"spanKm"`
}

type ContinentInfo struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Codes []string `json:"codes"`
}

type ScoringInfo struct {
	WorldSpanKm   float64 `json:"worldSpanKm"`
	WorldHalfKm   float64 `json:"worldHalfKm"`
	MinimumHalfKm float64 `json:"minimumHalfKm"`
	PerfectMeters float64 `json:"perfectMeters"`
}

type CountryCatalog struct {
	Countries  []CountryInfo   `json:"countries"`
	Continents []ContinentInfo `json:"continents"`
	Scoring    ScoringInfo     `json:"scoring"`
}

// Published once at startup with countryDB, before serving requests. Never
// expose the backing slices/maps; lobby inputs must not change sampling data.
var catalog CountryCatalog
var catalogByCode map[string]CountryInfo

func GetCountryCatalog() (CountryCatalog, error) {
	if len(catalog.Countries) == 0 {
		return CountryCatalog{}, ErrLocationData
	}
	out := catalog
	out.Countries = append([]CountryInfo{}, catalog.Countries...)
	out.Continents = append([]ContinentInfo{}, catalog.Continents...)
	for i := range out.Continents {
		out.Continents[i].Codes = append([]string{}, out.Continents[i].Codes...)
	}
	return out, nil
}

func buildCountryCatalog(countries map[string]*country, codes []string) (CountryCatalog, map[string]CountryInfo, error) {
	var definitions struct {
		Continents []ContinentInfo `json:"continents"`
	}
	b, err := os.ReadFile("assets/continents.json")
	if err != nil {
		return CountryCatalog{}, nil, err
	}
	if err := json.Unmarshal(b, &definitions); err != nil {
		return CountryCatalog{}, nil, err
	}
	validIDs := map[string]string{"AF": "Africa", "AS": "Asia", "EU": "Europe", "NA": "North America", "OC": "Oceania", "SA": "South America"}
	seenContinents := make(map[string]bool)
	placement := make(map[string]string, len(countries))
	for i, continent := range definitions.Continents {
		if validIDs[continent.ID] != continent.Name || continent.Name == "" || seenContinents[continent.ID] || len(continent.Codes) == 0 {
			return CountryCatalog{}, nil, errors.New("invalid continent definition")
		}
		seenContinents[continent.ID] = true
		for _, code := range continent.Codes {
			if countries[code] == nil || placement[code] != "" {
				return CountryCatalog{}, nil, fmt.Errorf("invalid or duplicate continent country: %s", code)
			}
			placement[code] = continent.ID
		}
		sort.Strings(definitions.Continents[i].Codes)
	}
	if len(placement) != len(countries) || len(seenContinents) != len(validIDs) {
		return CountryCatalog{}, nil, errors.New("incomplete continent definitions")
	}
	sort.Slice(definitions.Continents, func(i, j int) bool { return definitions.Continents[i].ID < definitions.Continents[j].ID })
	out := CountryCatalog{
		Countries: make([]CountryInfo, 0, len(codes)), Continents: definitions.Continents,
		Scoring: ScoringInfo{WorldSpanKm: WorldSpanKm, WorldHalfKm: LegacyHalfKm(100), MinimumHalfKm: MinimumHalfKm, PerfectMeters: PerfectMeters},
	}
	byCode := make(map[string]CountryInfo, len(codes))
	for _, code := range codes {
		c := countries[code]
		center, span, err := countryExtent(c.Areas)
		if err != nil {
			return CountryCatalog{}, nil, fmt.Errorf("invalid country geometry %s: %w", code, err)
		}
		info := CountryInfo{Code: code, Name: c.Name, Continent: placement[code], Center: center, SpanKm: span}
		out.Countries = append(out.Countries, info)
		byCode[code] = info
	}
	return out, byCode, nil
}

// countryExtent takes O(V log V) time and O(V) space, not vertex-pair distance
// work. Remove the largest circular longitude gap, then use the resulting
// dateline-aware rectangle's midpoint. Twice its maximum spherical radius is a
// conservative diameter, including edges/interior and Street View search radii.
func countryExtent(areas MultiPolygon) (Coords, float64, error) {
	longitudes := []float64{}
	minLat, maxLat := 90.0, -90.0
	maxRadiusKm := 0.0
	for _, polygon := range areas.SearchArea {
		if polygon == nil || polygon.Size <= 0 || polygon.Radius <= 0 || len(polygon.Rings) == 0 {
			return Coords{}, 0, ErrLocationData
		}
		maxRadiusKm = max(maxRadiusKm, float64(polygon.Radius)/1000)
		for _, ring := range polygon.Rings {
			if len(ring) < 3 {
				return Coords{}, 0, ErrLocationData
			}
			for _, p := range ring {
				if !(Coords{Lat: p[1], Lng: p[0]}).Valid() {
					return Coords{}, 0, ErrLocationData
				}
				longitudes = append(longitudes, math.Mod(p[0]+360, 360))
				minLat, maxLat = min(minLat, p[1]), max(maxLat, p[1])
			}
			// A closed last point is optional (the sampler closes it). Reject
			// zero-area rings; unwrap each edge across the dateline first.
			area, x := 0.0, 0.0
			for i, p := range ring {
				q := ring[(i+1)%len(ring)]
				nextX := x + wrapLongitude(q[0]-p[0])
				area += x*(q[1]-ring[0][1]) - nextX*(p[1]-ring[0][1])
				x = nextX
			}
			if area == 0 || math.IsNaN(area) || math.IsInf(area, 0) {
				return Coords{}, 0, ErrLocationData
			}
		}
	}
	if len(longitudes) == 0 {
		return Coords{}, 0, ErrLocationData
	}
	sort.Float64s(longitudes)
	gap, start := -1.0, 0.0
	for i, longitude := range longitudes {
		next := longitudes[(i+1)%len(longitudes)]
		if i == len(longitudes)-1 {
			next += 360
		}
		if next-longitude > gap {
			gap, start = next-longitude, next
		}
	}
	halfWidth := (360 - gap) / 2
	center := Coords{Lat: (minLat + maxLat) / 2, Lng: wrapLongitude(start + halfWidth)}
	edgeLng := wrapLongitude(center.Lng + halfWidth)
	radiusKm := max(CalcDistance(center, Coords{Lat: minLat, Lng: edgeLng}), CalcDistance(center, Coords{Lat: maxLat, Lng: edgeLng})) / 1000
	// At widths >180 degrees the farthest latitude may lie inside an edge,
	// not at a corner. Check extrema of A*sin(lat)+B*cos(lat) as well.
	a := math.Sin(center.Lat * math.Pi / 180)
	b := math.Cos(center.Lat*math.Pi/180) * math.Cos(halfWidth*math.Pi/180)
	stationary := math.Atan2(a, b)
	for _, latitude := range []float64{stationary - math.Pi, stationary, stationary + math.Pi} {
		lat := latitude * 180 / math.Pi
		if lat >= minLat && lat <= maxLat {
			radiusKm = max(radiusKm, CalcDistance(center, Coords{Lat: lat, Lng: edgeLng})/1000)
		}
	}
	span := min(WorldSpanKm, 2*(radiusKm+maxRadiusKm))
	if !center.Valid() || span <= 0 || math.IsNaN(span) || math.IsInf(span, 0) {
		return Coords{}, 0, ErrLocationData
	}
	return center, span, nil
}

func wrapLongitude(longitude float64) float64 {
	return math.Mod(math.Mod(longitude+180, 360)+360, 360) - 180
}

// NormalizeCountryCodes preserves nil (omitted update) versus [] (world),
// rejects unknown/non-uppercase codes, and returns sorted detached unique codes.
func NormalizeCountryCodes(codes []string) ([]string, error) {
	if len(codes) > MaxCountryCodes {
		return nil, errors.New("INVALID_LOBBY_SETTINGS")
	}
	if codes == nil {
		return nil, nil
	}
	out := make([]string, 0, len(codes))
	seen := make(map[string]bool, len(codes))
	for _, code := range codes {
		if _, ok := catalogByCode[code]; !ok {
			return nil, errors.New("INVALID_COUNTRY_CODE")
		}
		if !seen[code] {
			seen[code] = true
			out = append(out, code)
		}
	}
	sort.Strings(out)
	return out, nil
}

// SelectionSpanKm combines country extents and their separation. Callers get
// the same result using only the public catalog; at most 256 unique countries.
func SelectionSpanKm(codes []string) (float64, error) {
	codes, err := NormalizeCountryCodes(codes)
	if err != nil {
		return 0, err
	}
	if len(catalogByCode) == 0 {
		return 0, ErrLocationData
	}
	if len(codes) == 0 || len(codes) == len(catalogByCode) {
		return WorldSpanKm, nil
	}
	span := 0.0
	for i, code := range codes {
		a := catalogByCode[code]
		span = max(span, a.SpanKm)
		for _, other := range codes[:i] {
			b := catalogByCode[other]
			span = max(span, CalcDistance(a.Center, b.Center)/1000+(a.SpanKm+b.SpanKm)/2)
		}
	}
	return min(WorldSpanKm, span), nil
}

func AutomaticHalfKm(codes []string) (float64, error) {
	span, err := SelectionSpanKm(codes)
	if err != nil {
		return 0, err
	}
	worldHalf := LegacyHalfKm(100)
	if span == WorldSpanKm {
		return worldHalf, nil
	}
	return max(MinimumHalfKm, worldHalf*span/WorldSpanKm), nil
}
