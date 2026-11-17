package game

import (
	"errors"
	"math"

	"github.com/geobattles/geobattles-backend/pkg/defaults"
	"github.com/geobattles/geobattles-backend/pkg/logic"
)

// Zero/omitted numeric fields keep their existing default/no-update meaning.
// Reject unsafe supplied values before creating a hub or changing any fields.
func ValidateLobbyConf(conf LobbyConf) error {
	if (conf.Mode != 0 && conf.Mode != 1 && conf.Mode != 2) ||
		conf.MaxPlayers < 0 || conf.MaxPlayers > len(colorList) ||
		conf.NumAttempt < 0 || conf.NumAttempt > 100 ||
		conf.NumRounds < 0 || conf.NumRounds > 100 ||
		conf.RoundTime < 0 || conf.RoundTime > 3600 || (conf.RoundTime > 0 && conf.RoundTime < 5) ||
		(conf.ScoreFactor != 0 && (conf.ScoreFactor < defaults.ScoreFactorLow || conf.ScoreFactor > defaults.ScoreFactorHigh)) ||
		(conf.ScoringMode != "" && conf.ScoringMode != ScoringAuto && conf.ScoringMode != ScoringManual) ||
		math.IsNaN(conf.ScoreScaleKm) || math.IsInf(conf.ScoreScaleKm, 0) ||
		(conf.Powerups != nil && len(*conf.Powerups) != 2) {
		return errors.New("INVALID_LOBBY_SETTINGS")
	}
	_, err := logic.NormalizeCountryCodes(conf.CCList)
	return err
}
