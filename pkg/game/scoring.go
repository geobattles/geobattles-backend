package game

import (
	"math"
	"slices"

	"github.com/geobattles/geobattles-backend/pkg/logic"
)

const (
	ScoringAuto   = "auto"
	ScoringManual = "manual"
)

// normalizeConf returns a detached, complete configuration and sampling weight.
// All callers validate before mutation; scoreScaleKm is output, never authority.
func normalizeConf(c LobbyConf) (LobbyConf, float64, error) {
	if err := ValidateLobbyConf(c); err != nil {
		return LobbyConf{}, 0, err
	}
	c = defaultConf(c)
	codes, err := logic.NormalizeCountryCodes(c.CCList)
	if err != nil {
		return LobbyConf{}, 0, err
	}
	c.CCList = codes
	size, err := logic.SumCCListSize(c.CCList)
	if err != nil {
		return LobbyConf{}, 0, err
	}
	if c.ScoringMode == "" {
		c.ScoringMode = ScoringAuto
	}
	switch {
	case c.Mode == 2:
		// Country Battle has its unchanged time/country scoring, not a curve.
		c.ScoreScaleKm = 0
	case c.ScoringMode == ScoringManual:
		c.ScoreScaleKm = logic.LegacyHalfKm(c.ScoreFactor)
	default:
		c.ScoreScaleKm, err = logic.AutomaticHalfKm(c.CCList)
		if err != nil {
			return LobbyConf{}, 0, err
		}
	}
	return c, size, nil
}

// Inputs are already sorted/deduplicated. Explicit all-catalog and [] are the
// same world region; reordering/duplicates do not reset a manual policy.
func sameRegion(a, b []string) bool {
	worldA := len(a) == 0 || len(a) == len(logic.CountryList)
	worldB := len(b) == 0 || len(b) == len(logic.CountryList)
	return (worldA && worldB) || slices.Equal(a, b)
}

func scoreAutomatic(distanceMeters, halfKm float64) int {
	score := int(5000 * math.Pow(0.5, max(0, distanceMeters-logic.PerfectMeters)/(halfKm*1000)))
	return min(score, 5000)
}

func (c LobbyConf) scoreDistance(distanceMeters float64) int {
	if c.ScoringMode == ScoringManual {
		return scoreDistance(distanceMeters, float64(c.ScoreFactor))
	}
	return scoreAutomatic(distanceMeters, c.ScoreScaleKm)
}
