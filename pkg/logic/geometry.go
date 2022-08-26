// based on https://github.com/paulmach/orb
package logic

import (
	"math"
)

// structs representing country polygons in geojson format
// 1 country can be made of multiple polygons, each polygon has size and radius
// and can have multiple rings. first ring is the outer polygon, the rest are inner holes.
// each ring is made of multiple points, each point is a coordinate pair

type Point [2]float64
type Ring []Point
type Polygon struct {
	Size   int
	Radius int
	Rings  []Ring
}
type MultiPolygon struct {
	InnerSize  int
	SearchArea map[string]*Polygon
}

// bounding rectangle defined by two points
type Bound struct {
	Min, Max Point
}

// checks if point is within a bound
func (b Bound) Contains(point Point) bool {
	if point[1] < b.Min[1] || b.Max[1] < point[1] {
		return false
	}

	if point[0] < b.Min[0] || b.Max[0] < point[0] {
		return false
	}

	return true
}

// extends existing bound to include point
func (b Bound) Extend(point Point) Bound {
	// if point is already within bound just return existing bound
	if b.Contains(point) {
		return b
	}

	return Bound{
		Min: Point{
			math.Min(b.Min[0], point[0]),
			math.Min(b.Min[1], point[1]),
		},
		Max: Point{
			math.Max(b.Max[0], point[0]),
			math.Max(b.Max[1], point[1]),
		},
	}
}

// returns bound around ring
func (mp Ring) Bound() Bound {
	if len(mp) == 0 {
		return Bound{Min: Point{1, 1}, Max: Point{-1, -1}}
	}

	// initialize bound and extend it for each point on ring
	b := Bound{mp[0], mp[0]}
	for _, p := range mp {
		b = b.Extend(p)
	}

	return b
}

// checks if a point is inside MultiPolygon
func MultiPolyContains(mp map[string]*Polygon, pt Point) bool {
	for _, p := range mp {
		if polygonContains(p.Rings, pt) {
			return true
		}
	}
	return false
}

// checks if a point is inside Polygon
func polygonContains(p []Ring, pt Point) bool {
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
func ringContains(r Ring, pt Point) bool {
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
func rayIntersect(p, s, e Point) (intersects, on bool) {
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
