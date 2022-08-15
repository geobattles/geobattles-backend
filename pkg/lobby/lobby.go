package lobby

import "fmt"

type Lobby struct {
	Name       string   `json:"name"`
	ID         string   `json:"id"`
	MaxPlayers int      `json:"maxPlayers"`
	NumPlayers int      `json:"numPlayers"`
	PlayerList []string `json:"playerList"`
}

var LobbyList = []Lobby{
	{Name: "prvi lobby", ID: "U4YPR6", MaxPlayers: 8, NumPlayers: 0, PlayerList: nil},
	{Name: "LOBBY #2", ID: "8CKXRG", MaxPlayers: 6, NumPlayers: 0, PlayerList: nil},
}

func AddPlayerToLobby(LobbyList []Lobby, client string, lobbyID string) {
	for i := range LobbyList {
		if LobbyList[i].ID == lobbyID {
			fmt.Println("lobby matches adding name ", client)
			LobbyList[i].PlayerList = append(LobbyList[i].PlayerList, client)
			fmt.Println("all lobby list ", LobbyList)
			break
		}
	}
}

func RemovePlayerFromLobby(LobbyList []Lobby, client string, lobbyID string) {
	for i := range LobbyList {
		if LobbyList[i].ID == lobbyID {
			fmt.Println("lobby matches adding name ", client)
			for index, value := range LobbyList[i].PlayerList {
				if value == client {
					LobbyList[i].PlayerList = append(LobbyList[i].PlayerList[:index], LobbyList[i].PlayerList[index+1:]...)
					fmt.Println("all lobby list ", LobbyList)
					break
				}
			}

		}
	}
}
