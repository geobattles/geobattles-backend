package lobby

import (
	"example/web-service-gin/pkg/logic"
	"fmt"
)

type Lobby struct {
	Name            string                       `json:"name"`
	ID              string                       `json:"id"`
	MaxPlayers      int                          `json:"maxPlayers"`
	NumPlayers      int                          `json:"numPlayers"`
	PlayerList      []string                     `json:"playerList"`
	GameActive      bool                         `json:"gameActive"`
	CurrentLocation logic.Coordinates            `json:"currentLocation"`
	CurrentRound    int                          `json:"currentRound"`
	Results         map[int]map[string][]float64 `json:"results"`
}

var LobbyList = []Lobby{
	{Name: "prvi lobby", ID: "U4YPR6", MaxPlayers: 8, NumPlayers: 0, PlayerList: nil, Results: make(map[int]map[string][]float64)},
	{Name: "LOBBY #2", ID: "8CKXRG", MaxPlayers: 6, NumPlayers: 0, PlayerList: nil, Results: make(map[int]map[string][]float64)},
}

func AddPlayerToLobby(LobbyList []Lobby, client string, lobbyID string) {
	for i := range LobbyList {
		if LobbyList[i].ID == lobbyID {
			fmt.Println("lobby matches adding name ", client)
			LobbyList[i].PlayerList = append(LobbyList[i].PlayerList, client)
			LobbyList[i].NumPlayers = len(LobbyList[i].PlayerList)
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
					LobbyList[i].NumPlayers = len(LobbyList[i].PlayerList)
					break
				}
			}

		}
	}
}
func MarkGameActive(lobbyID string) {
	fmt.Println("req mark game started")
	for i := range LobbyList {
		if LobbyList[i].ID == lobbyID {
			fmt.Println("lobby found, marking true ", LobbyList[i].ID)
			LobbyList[i].GameActive = true
			break
		}
	}
}

func UpdateCurrentLocation(lobbyID string, location logic.Coordinates) {
	fmt.Println("req mark game started")
	for i := range LobbyList {
		if LobbyList[i].ID == lobbyID {
			fmt.Println("lobby found, marking true ", LobbyList[i].ID)
			LobbyList[i].CurrentLocation = location
			break
		}
	}
}

func CalculateDistance(lobbyID string, userLocation logic.Coordinates) float64 {
	fmt.Println("req mark game started")

	for i := range LobbyList {
		if LobbyList[i].ID == lobbyID {
			return logic.CalcDistance(LobbyList[i].CurrentLocation, userLocation)
		}
	}
	return 99999
}

func AddToResults(lobbyID string, clientID string, result float64) {
	for i := range LobbyList {
		if LobbyList[i].ID == lobbyID {
			fmt.Println("pred lobby z dodanim rezultatom", LobbyList[i])
			fmt.Println("pred lobby samo map", LobbyList[i].Results[0][clientID])
			if LobbyList[i].Results[0] == nil {
				fmt.Println("mapa se ne obstaja, ustvarjam")
				LobbyList[i].Results[0] = make(map[string][]float64)
			}
			LobbyList[i].Results[0][clientID] = append(LobbyList[i].Results[0][clientID], result)
			fmt.Println("lobby z dodanim rezultatom", LobbyList[i])
		}
	}
}
