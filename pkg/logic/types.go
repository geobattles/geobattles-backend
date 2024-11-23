package logic

type Coords struct {
	Lat float64 `json:"lat,omitempty"`
	Lng float64 `json:"lng,omitempty"`
}

// response from google maps metadata api
type ApiMetaResponse struct {
	Loc    Coords `json:"location"`
	Status string `json:"status"`
}
