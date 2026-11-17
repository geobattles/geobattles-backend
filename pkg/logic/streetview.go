package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// These errors are safe socket error types. Never wrap a transport error (which
// can contain the key-bearing request URL) or return Google's error_message.
var (
	ErrStreetViewNotFound   = errors.New("LOCATION_NOT_FOUND")
	ErrProviderDenied       = errors.New("LOCATION_PROVIDER_DENIED")
	ErrProviderQuota        = errors.New("LOCATION_PROVIDER_QUOTA")
	ErrProviderUnavailable  = errors.New("LOCATION_PROVIDER_UNAVAILABLE")
	ErrProviderResponse     = errors.New("LOCATION_PROVIDER_INVALID_RESPONSE")
	ErrProviderUnconfigured = errors.New("LOCATION_PROVIDER_UNCONFIGURED")
	ErrLocationTimeout      = errors.New("LOCATION_GENERATION_TIMEOUT")
	ErrLocationData         = errors.New("LOCATION_DATA_INVALID")
)

const metadataTimeout = 2 * time.Second
const maxMetadataBytes = 64 << 10

var metadataClient = &http.Client{
	Timeout: metadataTimeout,
	// Do not forward a key-bearing request through a provider redirect.
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

// CheckStreetViewExists performs exactly one bounded metadata request. Only an
// OK response with a complete, valid location can succeed; (0,0) is not a
// failure sentinel. RndLocation owns the shared retry/sampling budget.
func CheckStreetViewExists(ctx context.Context, loc Point, radius int) (Coords, error) {
	if !(Coords{Lat: loc[1], Lng: loc[0]}).Valid() || radius <= 0 {
		return Coords{}, ErrLocationData
	}
	key := os.Getenv("GMAPS_API")
	if key == "" {
		return Coords{}, ErrProviderUnconfigured
	}
	query := url.Values{
		"location": {fmt.Sprintf("%f,%f", loc[1], loc[0])},
		"key":      {key}, "radius": {fmt.Sprint(radius)}, "source": {"outdoor"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://maps.googleapis.com/maps/api/streetview/metadata?"+query.Encode(), nil)
	if err != nil {
		return Coords{}, ErrProviderUnavailable
	}
	res, err := metadataClient.Do(req)
	if err != nil {
		return Coords{}, ErrProviderUnavailable
	}
	if res == nil || res.Body == nil {
		return Coords{}, ErrProviderResponse
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden:
		return Coords{}, ErrProviderDenied
	case res.StatusCode == http.StatusTooManyRequests:
		return Coords{}, ErrProviderQuota
	case res.StatusCode == http.StatusRequestTimeout || res.StatusCode == 500 || res.StatusCode == 502 || res.StatusCode == 503 || res.StatusCode == 504:
		return Coords{}, ErrProviderUnavailable
	case res.StatusCode != http.StatusOK:
		return Coords{}, ErrProviderResponse
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxMetadataBytes+1))
	if err != nil {
		return Coords{}, ErrProviderUnavailable
	}
	if len(body) > maxMetadataBytes {
		return Coords{}, ErrProviderResponse
	}
	var response struct {
		Loc    *Coords `json:"location"`
		Status string  `json:"status"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return Coords{}, ErrProviderResponse
	}
	switch response.Status {
	case "OK":
		if response.Loc == nil || !response.Loc.Valid() {
			return Coords{}, ErrProviderResponse
		}
		return *response.Loc, nil
	case "ZERO_RESULTS":
		return Coords{}, ErrStreetViewNotFound
	case "UNKNOWN_ERROR":
		return Coords{}, ErrProviderUnavailable
	case "OVER_QUERY_LIMIT", "OVER_DAILY_LIMIT":
		return Coords{}, ErrProviderQuota
	case "REQUEST_DENIED":
		return Coords{}, ErrProviderDenied
	default:
		return Coords{}, ErrProviderResponse
	}
}
