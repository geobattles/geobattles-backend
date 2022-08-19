package logic

//var LastSentLoc Coordinates

type Coordinates struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
}

// response from google maps metadata api
type MetadataResponse struct {
	Location Coordinates `json:"location"`
	Status   string
}

type ClientReq struct {
	Command  string      `json:"command"`
	Location Coordinates `json:"location"`
}

type ResponseMsg struct {
	Status   string                       `json:"status"`
	Location Coordinates                  `json:"location,omitempty"`
	Room     string                       `json:"-"`
	Distance float64                      `json:"distance,omitempty"`
	Results  map[int]map[string][]float64 `json:"results,omitempty"`
}
