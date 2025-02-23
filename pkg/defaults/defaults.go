package defaults

const (
	Mode            = 1
	NumOfTries      = 3
	NumOfRounds     = 5
	MaxPlayers      = 10
	ScoreFactor     = 100
	ScoreFactorLow  = 1
	ScoreFactorHigh = 500
	RoundTime       = 90
)

// we cant use constant array/slice so we use this instead
func Powerups() *[]bool {
	return &[]bool{true, true}
}
func PlaceBonus() *bool {
	b := false
	return &b
}
func DynLives() *bool {
	b := false
	return &b
}
