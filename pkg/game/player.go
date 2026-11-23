package game

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/geobattles/geobattles-backend/pkg/logic"
	"github.com/geobattles/geobattles-backend/pkg/models"
	"github.com/geobattles/geobattles-backend/pkg/reverse"
	"github.com/geobattles/geobattles-backend/pkg/websocket"
)

func PlayerMessageHandler(c *websocket.Client, message []byte) {
	var req ClientReq
	parseErr := json.Unmarshal(message, &req)
	if parseErr != nil {
		// Recover independently valid envelope fields even when another field
		// (for example a fractional round) failed typed decoding.
		var fields map[string]json.RawMessage
		if json.Unmarshal(message, &fields) == nil {
			_ = json.Unmarshal(fields["command"], &req.Cmd)
			_ = json.Unmarshal(fields["gameId"], &req.GameID)
			_ = json.Unmarshal(fields["requestId"], &req.RequestID)
			req.Round = nil
			var round *int
			if json.Unmarshal(fields["round"], &round) == nil {
				req.Round = round
			}
		}
	}
	l, ok := c.Lobby.(*Lobby)
	if !ok || l == nil {
		sendRequestError(c, req.Cmd, "INVALID_GAME_STATE")
		return
	}
	var e lobbyEffects
	var selection *roundStart
	l.mu.Lock()
	// Every command (including disconnect) is conditional on this exact socket.
	if !l.ownsLocked(c) || c.Closed() {
		l.mu.Unlock()
		c.Close()
		return
	}
	if parseErr != nil {
		code := "INVALID_REQUEST"
		if errors.Is(parseErr, logic.ErrInvalidLocation) {
			code = "INVALID_LOCATION"
		}
		if req.Cmd == "use_powerup" || req.Cmd == "powerup_ready" || req.Cmd == "start" {
			l.powerupErrorLocked(&e, c, req, code)
		} else {
			l.errorLocked(&e, c, req.Cmd, code)
		}
		l.mu.Unlock()
		l.applyEffects(e)
		return
	}
	if (req.Cmd == "submit_location" || req.Cmd == "loc_to_cc") && (req.Loc == nil || !req.Loc.Valid()) {
		l.errorLocked(&e, c, req.Cmd, "INVALID_LOCATION")
		l.mu.Unlock()
		l.applyEffects(e)
		return
	}
	switch req.Cmd {
	case "leave":
		l.retainSeatLocked(&e, c, true)
	case "disconnect":
		l.leaveLocked(&e, c)
	case "setReady":
		if err := l.setReadyLocked(c.ID, req.Ready); err != nil {
			l.errorLocked(&e, c, req.Cmd, err.Error())
		} else {
			l.broadcastLocked(&e, models.ResponseBase{Status: "OK", Type: "UPDATED_LOBBY", Payload: models.ResponsePayload{Lobby: l.snapshotLocked()}})
		}
	case "update_lobby_settings":
		if err := l.updateConfLocked(c.ID, req.Conf); err != nil {
			l.errorLocked(&e, c, req.Cmd, err.Error())
		} else {
			l.broadcastLocked(&e, models.ResponseBase{Status: "OK", Type: "UPDATED_LOBBY", Payload: models.ResponsePayload{Lobby: l.snapshotLocked()}})
		}
	case "start":
		var err error
		if l.phase == PhaseMidRound {
			err = l.requestPowerupStartLocked(&e, c, req)
		} else {
			selection, err = l.beginStartLocked(c)
			if err == nil {
				l.broadcastPowerupLocked(&e)
			}
		}
		if err != nil {
			l.powerupErrorLocked(&e, c, req, err.Error())
		}
	case "use_powerup", "powerup_ready":
		if err := l.powerupCommandLocked(&e, c, req); err != nil {
			l.powerupErrorLocked(&e, c, req, err.Error())
		}
	case "submit_location":
		// A delayed timer callback cannot admit a guess after the deadline.
		if l.active && !time.Now().Before(l.deadline) {
			l.finalizeLocked(&e, l.roundGen)
		}
		if err := l.submitResultLocked(c.ID, *req.Loc); err != nil {
			l.errorLocked(&e, c, req.Cmd, err.Error())
		} else {
			results := l.rawResults[l.round][c.ID]
			last := results[len(results)-1]
			// Commit and enqueue the last guess BEFORE the entire end batch.
			l.broadcastLocked(&e, models.ResponseBase{Status: "OK", Type: "NEW_RESULT", Payload: models.ResponsePayload{GameID: l.gameID, User: c.ID, GuessRes: last, Round: l.round, Phase: l.phase}})
			if l.allFinishedLocked() {
				l.finalizeLocked(&e, l.roundGen)
			}
		}
	case "loc_to_cc":
		// Keep this reader's commands serial without holding up lobby timers
		// or other players during the polygon search.
		l.mu.Unlock()
		cc, err := reverse.ReverseGeocode(req.Loc.Lng, req.Loc.Lat)
		l.mu.Lock()
		if !l.ownsLocked(c) || c.Closed() {
			l.mu.Unlock()
			return
		}
		if err != nil {
			l.errorLocked(&e, c, req.Cmd, err.Error())
		} else {
			l.sendLocked(&e, c, models.ResponseBase{Status: "OK", Type: "CC", Payload: models.ResponsePayload{CC: cc, Polygon: append(json.RawMessage(nil), logic.PolyDB[cc]...)}})
		}
	default:
		l.errorLocked(&e, c, req.Cmd, "INVALID_COMMAND")
	}
	l.launchPowerupStartLocked(&e)
	l.mu.Unlock()
	l.applyEffects(e)
	if selection != nil {
		go l.completeStart(selection)
	}
}
