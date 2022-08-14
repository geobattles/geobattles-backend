package lobby

type Lobby struct {
	Name       string `json:"name"`
	ID         int    `json:"id"`
	MaxPlayers int    `json:"maxPlayers"`
	NumPlayers int    `json:"numPlayers"`
}
