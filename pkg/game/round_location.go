//go:build !matchfixture

package game

import "github.com/geobattles/geobattles-backend/pkg/logic"

func (work *roundStart) location() (logic.Coords, string, error) {
	return logic.RndLocationContext(work.ctx, work.countries, work.ccSize)
}
