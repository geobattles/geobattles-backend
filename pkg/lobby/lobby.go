package lobby

type Lobby struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	MaxPlayers int    `json:"maxPlayers"`
	NumPlayers int    `json:"numPlayers"`
}
