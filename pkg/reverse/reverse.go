package reverse

import (
	"errors"
	"math"
)

// returns country code of a given lan & lng
func ReverseGeocode(lng float64, lat float64) (string, error) {
	pt := point{lng, lat}

	for _, feature := range fullPolygons.Features {
		// skip full search if location is outside bounding box
		if lng < feature.Bbox[0] || lng > feature.Bbox[2] || lat < feature.Bbox[1] || lat > feature.Bbox[3] {
			continue
		}

		for _, polygon := range feature.Geometry.Coordinates {
			if polygonContains(polygon, pt) {
				return feature.Properties["cc"], nil
			}
		}
	}

	return "", errors.New("NO_COUNTRY")
}

func polygonContains(p []ring, pt point) bool {
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

func ringContains(r ring, pt point) bool {
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
func rayIntersect(p, s, e point) (intersects, on bool) {
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

type Bound struct {
	Min, Max point
}

// checks if point is within a bound
func (b Bound) Contains(p point) bool {
	if p[1] < b.Min[1] || b.Max[1] < p[1] {
		return false
	}

	if p[0] < b.Min[0] || b.Max[0] < p[0] {
		return false
	}

	return true
}

// extends existing bound to include point
func (b Bound) Extend(p point) Bound {
	// if point is already within bound just return existing bound
	if b.Contains(p) {
		return b
	}

	return Bound{
		Min: point{
			math.Min(b.Min[0], p[0]),
			math.Min(b.Min[1], p[1]),
		},
		Max: point{
			math.Max(b.Max[0], p[0]),
			math.Max(b.Max[1], p[1]),
		},
	}
}

func (mp ring) Bound() Bound {
	if len(mp) == 0 {
		return Bound{Min: point{1, 1}, Max: point{-1, -1}}
	}

	// initialize bound and extend it for each point on ring
	b := Bound{mp[0], mp[0]}
	for _, p := range mp {
		b = b.Extend(p)
	}

	return b
}
