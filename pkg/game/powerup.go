package game

import (
	"errors"
	"log/slog"
	"sort"
)

// The active ledger, not the compatibility powerLog, is the settlement authority.
// Status transitions make a second invocation harmless as well as finalization's
// existing once-only generation guard.
func (l *Lobby) processPowerups() error {
	for i := range l.powerEffects {
		r := &l.powerEffects[i]
		if r.Round != l.round || r.Status != "ACTIVE" {
			continue
		}
		source := l.leases[r.Source]
		if source == nil || source.present != r.sourceMember {
			r.Status = "CANCELLED"
			r.Reason = "SOURCE_LEFT"
			continue
		}
		s := l.endResults[l.round][r.Source]
		if r.Type == 0 {
			if s != nil {
				s.DoubleScore = s.BaseScore
			}
			r.Status = "APPLIED"
			continue
		}
		target := l.leases[r.Target]
		if target == nil || target.present != r.targetMember {
			r.Status = "REFUNDED"
			r.Reason = "TARGET_LEFT"
			if p := l.players[r.Source]; p != nil && len(p.Powerups) == 2 {
				p.Powerups[1] = true
			}
			continue
		}
		t := l.endResults[l.round][r.Target]
		winner := duelWinner(l.conf.Mode, s, t)
		r.Status = "APPLIED"
		r.Outcome = "DRAW"
		if winner != 0 && s != nil && t != nil {
			s.DuelScore += 1000 * winner
			t.DuelScore -= 1000 * winner
			if winner > 0 {
				r.Outcome = "SOURCE_WON"
			} else {
				r.Outcome = "TARGET_WON"
			}
		}
	}
	return nil
}

// +1 source wins, -1 target wins, 0 draw. A real zero-distance guess is valid.
func duelWinner(mode int, s, t *Result) int {
	valid := func(r *Result) bool { return r != nil && r.Attempt > 0 && (mode != 2 || r.CC == "XX") }
	sv, tv := valid(s), valid(t)
	if !sv && !tv {
		return 0
	}
	if !tv {
		return 1
	}
	if !sv {
		return -1
	}
	if mode == 2 {
		if s.Time < t.Time {
			return 1
		}
		if s.Time > t.Time {
			return -1
		}
		return 0
	}
	if s.Dist < t.Dist {
		return 1
	}
	if s.Dist > t.Dist {
		return -1
	}
	return 0
}

// add bonus points based on player placement. first player gets 30% bonus, second 20, third 10
// depends on the number of players; with 2 players only first gets 10%
func (l *Lobby) processBonus() error {
	if !*l.conf.PlaceBonus {
		slog.Debug("Bonus disabled")
		return errors.New("BONUS_DISABLED")
	}
	var playerOrder []string
	for id := range l.endResults[l.round] {
		// Retaining an absent seat in the results must not inflate placement
		// bonuses for rounds the player never entered.
		if l.roundEntrants[id] {
			playerOrder = append(playerOrder, id)
		}
	}
	sort.SliceStable(playerOrder, func(i, j int) bool {
		iEmpty := l.endResults[l.round][playerOrder[i]].Attempt == 0
		jEmpty := l.endResults[l.round][playerOrder[j]].Attempt == 0
		if iEmpty || jEmpty {
			return !iEmpty && jEmpty // both placeholders compare false
		}
		switch l.conf.Mode {
		case 2:
			return l.endResults[l.round][playerOrder[i]].Time < l.endResults[l.round][playerOrder[j]].Time
		default:
			return l.endResults[l.round][playerOrder[i]].Dist < l.endResults[l.round][playerOrder[j]].Dist
		}
	})
	slog.Debug("Player order", "playerOrder", playerOrder)
	switch num := len(playerOrder); {
	case num == 2:
		l.endResults[l.round][playerOrder[0]].PlaceScore = int(float64(l.endResults[l.round][playerOrder[0]].BaseScore) * 0.1)
	case num == 3:
		l.endResults[l.round][playerOrder[0]].PlaceScore = int(float64(l.endResults[l.round][playerOrder[0]].BaseScore) * 0.2)
		l.endResults[l.round][playerOrder[1]].PlaceScore = int(float64(l.endResults[l.round][playerOrder[1]].BaseScore) * 0.1)
	case num >= 4:
		l.endResults[l.round][playerOrder[0]].PlaceScore = int(float64(l.endResults[l.round][playerOrder[0]].BaseScore) * 0.3)
		l.endResults[l.round][playerOrder[1]].PlaceScore = int(float64(l.endResults[l.round][playerOrder[1]].BaseScore) * 0.2)
		l.endResults[l.round][playerOrder[2]].PlaceScore = int(float64(l.endResults[l.round][playerOrder[2]].BaseScore) * 0.1)
	}
	return nil
}

// processes total score for each player (total score = base + place + double + duel)
func (l *Lobby) processTotal() error {
	for player, result := range l.endResults[l.round] {
		roundTotal := result.BaseScore + result.PlaceScore + result.DoubleScore + result.DuelScore
		l.endResults[l.round][player].Total = roundTotal
		l.totalResults[player].Total += roundTotal
	}
	return nil
}
