package game

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/geobattles/geobattles-backend/pkg/db"
	"github.com/geobattles/geobattles-backend/pkg/models"
	"github.com/geobattles/geobattles-backend/pkg/websocket"
)

type peerClose struct {
	code   int
	reason string
}

type lobbyEffects struct {
	closePeers map[*websocket.Client]peerClose
	closeLobby bool
	save       *models.Game
	selection  *roundStart
}

func (e *lobbyEffects) close(c *websocket.Client) {
	if e.closePeers == nil {
		e.closePeers = make(map[*websocket.Client]peerClose)
	}
	e.closePeers[c] = peerClose{}
}

func (e *lobbyEffects) closeWith(c *websocket.Client, code int, reason string) {
	e.close(c)
	e.closePeers[c] = peerClose{code: code, reason: reason}
}

// Encode under the transaction, then only immutable bytes cross transport.
// Queue failure is a disconnect, not a silently lost state transition.
func (l *Lobby) broadcastLocked(e *lobbyEffects, message models.ResponseBase) {
	if l.closed {
		return
	}
	l.seq++
	message.Seq = l.seq
	encoded, err := json.Marshal(message)
	if err != nil {
		slog.Error("Cannot encode lobby event", "type", message.Type)
		l.closeLocked(e)
		return
	}
	for _, c := range l.hub.Broadcast(encoded) {
		e.close(c)
	}
}

func (l *Lobby) sendLocked(e *lobbyEffects, c *websocket.Client, message models.ResponseBase) {
	if c == nil || l.closed {
		return
	}
	l.seq++
	message.Seq = l.seq
	encoded, err := json.Marshal(message)
	if err != nil || !l.hub.Send(c, encoded) {
		e.close(c)
	}
}

func (l *Lobby) errorLocked(e *lobbyEffects, c *websocket.Client, command, code string) {
	l.sendLocked(e, c, requestError(command, code))
}

// Called only AFTER l.mu is released. No registry/lobby lock nesting, socket
// close/callback or persistence work inside a game/transport critical section.
func (l *Lobby) applyEffects(e lobbyEffects) {
	if e.selection != nil {
		go l.completeStart(e.selection)
	}
	if e.closeLobby {
		deleteLobbyIfSame(l)
		l.hub.Close()
	}
	for c, close := range e.closePeers {
		if close.code == 0 {
			c.Close()
		} else {
			c.CloseWithCode(close.code, close.reason)
		}
	}
	if e.save != nil {
		// At most 100 rounds x 12 best results, no raw guess history. One attempt
		// per completed game; deadline bounds waiting, not a durable retry queue.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if db.DB == nil || db.DB.WithContext(ctx).Create(e.save).Error != nil {
			slog.Error("Could not save completed game results")
		} else {
			slog.Info("Saved game results", "gameID", e.save.ID)
		}
	}
}

func (l *Lobby) persistenceSnapshotLocked() *models.Game {
	game := &models.Game{Timestamp: time.Now(), Rounds: make([]models.Round, l.round)}
	for index := 1; index <= l.round; index++ {
		round := &game.Rounds[index-1]
		round.Loc = l.locations[index-1]
		round.Results = make([]models.Result, 0, len(l.endResults[index]))
		for id, r := range l.endResults[index] {
			round.Results = append(round.Results, models.Result{UserID: id, Loc: r.Loc, Dist: r.Dist, Total: r.Total})
		}
	}
	return game
}
