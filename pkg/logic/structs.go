package logic

type Coordinates struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
}

type MetadataResponse struct {
	Location Coordinates `json:"location"`
	Status   string
}

type ClientReq struct {
	Message string `json:"message"`
}

type ResponseMsg struct {
	Status   string      `json:"status"`
	Location Coordinates `json:"location"`
}
