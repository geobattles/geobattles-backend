//go:build matchfixture

package game

import (
	"errors"

	"github.com/geobattles/geobattles-backend/pkg/logic"
)

// The fixture is indexed by the requested round, including reconnects and replays.
// Normal server binaries do not contain these locations.
func (work *roundStart) location() (logic.Coords, string, error) {
	if err := work.ctx.Err(); err != nil {
		return logic.Coords{}, "", err
	}
	switch work.round {
	case 1:
		return logic.Coords{Lat: 46, Lng: 15}, "SI", nil
	case 2:
		return logic.Coords{Lat: 52, Lng: 5}, "NL", nil
	default:
		return logic.Coords{}, "", errors.New("match fixture supports two rounds")
	}
}
