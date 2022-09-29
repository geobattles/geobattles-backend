package logic

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

type status struct {
	Message string `json:"message"`
	Value   int    `json:"value"`
}

type apiResponse struct {
	CountryCode string `json:"countryCode"`
	Status      status `json:"status"`
}

func LocToCC(loc Coords) (string, error) {
	fmt.Println(fmt.Sprintf("http://api.geonames.org/countryCode?lat=%f&lng=%f&username=%s&type=JSON", loc.Lat, loc.Lng, os.Getenv("GEONAMES_USER")))
	res, err := http.Get(fmt.Sprintf("http://api.geonames.org/countryCode?lat=%f&lng=%f&username=%s&type=JSON", loc.Lat, loc.Lng, os.Getenv("GEONAMES_USER")))
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}
	var response apiResponse
	json.Unmarshal(body, &response)
	fmt.Println(response.Status.Value, response.CountryCode)
	if response.Status.Value != 0 {
		fmt.Println("GEO reverse err", response.Status.Message)
		return "", errors.New("NO_COUNTRY")
	}
	return response.CountryCode, nil
}
