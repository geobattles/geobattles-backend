package defaults

const (
	NumOfTries      = 3
	NumOfRounds     = 5
	MaxPlayers      = 10
	ScoreFactor     = 100
	ScoreFactorLow  = 1
	ScoreFactorHigh = 500
	RoundTime       = 60
)

// we cant use constant array/slice so we use this instead
func Powerups() *[]bool {
	return &[]bool{true, true}
}
func PlaceBonus() *bool {
	b := true
	return &b
}
func DynLives() *bool {
	b := true
	return &b
}
