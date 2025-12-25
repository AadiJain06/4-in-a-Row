package game

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	StatusWaiting  = "waiting"
	StatusActive   = "active"
	StatusFinished = "finished"
)

type GameState struct {
	ID         string
	Board      Board
	Status     string
	Winner     string
	StartedAt  time.Time
	EndedAt    time.Time
	Turn       int
	LastMoveAt time.Time
	Players    map[string]*Player
	Bot        *Bot
}

type Player struct {
	Username string
	Slot     int
	IsBot    bool
}

type Manager struct {
	mu             sync.RWMutex
	waiting        *Player
	games          map[string]*GameState
	userToGame     map[string]string
	reconnectAfter time.Duration
	onFinish       func(*GameState)
}

type Move struct {
	Username string
	GameID   string
	Column   int
}

func NewManager(reconnectWindow time.Duration, onFinish func(*GameState)) *Manager {
	return &Manager{
		games:          make(map[string]*GameState),
		userToGame:     make(map[string]string),
		reconnectAfter: reconnectWindow,
		onFinish:       onFinish,
	}
}

func (m *Manager) AssignPlayer(username string) (*GameState, *Player, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Rejoin existing game if present.
	if gid, ok := m.userToGame[username]; ok {
		if g, exists := m.games[gid]; exists && g.Status != StatusFinished {
			p := g.Players[username]
			return g, p, false
		}
	}

	player := &Player{Username: username, Slot: CellP1}
	if m.waiting == nil {
		m.waiting = player
		return nil, player, true
	}

	// Start new game.
	opponent := m.waiting
	m.waiting = nil
	game := &GameState{
		ID:        uuid.NewString(),
		Status:    StatusActive,
		Turn:      CellP1,
		StartedAt: time.Now(),
		LastMoveAt: time.Now(),
		Players: map[string]*Player{
			opponent.Username: opponent,
			player.Username:   {Username: username, Slot: CellP2},
		},
	}
	m.games[game.ID] = game
	m.userToGame[player.Username] = game.ID
	m.userToGame[opponent.Username] = game.ID
	return game, game.Players[username], false
}

func (m *Manager) StartBotGame(human string) *GameState {
	m.mu.Lock()
	defer m.mu.Unlock()

	if gid, ok := m.userToGame[human]; ok {
		if g, exists := m.games[gid]; exists && g.Status != StatusFinished {
			return g
		}
	}

	humanPlayer := &Player{Username: human, Slot: CellP1}
	bot := NewBot(CellP2)
	game := &GameState{
		ID:        uuid.NewString(),
		Status:    StatusActive,
		Turn:      CellP1,
		StartedAt: time.Now(),
		LastMoveAt: time.Now(),
		Players: map[string]*Player{
			human: humanPlayer,
			"bot": {Username: "bot", Slot: CellP2, IsBot: true},
		},
		Bot: bot,
	}
	m.games[game.ID] = game
	m.userToGame[human] = game.ID
	return game
}

func (m *Manager) HandleMove(move Move) (MoveResult, *GameState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	game, ok := m.games[move.GameID]
	if !ok {
		return MoveResult{}, nil, ErrInvalidTurn
	}
	if game.Status == StatusFinished {
		return MoveResult{}, game, ErrGameFinished
	}
	player, ok := game.Players[move.Username]
	if !ok {
		return MoveResult{}, game, ErrInvalidTurn
	}
	if game.Turn != player.Slot {
		return MoveResult{}, game, ErrInvalidTurn
	}
	res, err := game.Board.ApplyMove(move.Column, player.Slot)
	if err != nil {
		return MoveResult{}, game, err
	}
	game.LastMoveAt = time.Now()
	if res.Winner != 0 {
		game.Status = StatusFinished
		game.Winner = move.Username
		game.EndedAt = time.Now()
		if m.onFinish != nil {
			go m.onFinish(game)
		}
	} else if res.IsDraw {
		game.Status = StatusFinished
		game.EndedAt = time.Now()
		if m.onFinish != nil {
			go m.onFinish(game)
		}
	} else {
		if game.Turn == CellP1 {
			game.Turn = CellP2
		} else {
			game.Turn = CellP1
		}
	}
	return res, game, nil
}

func (m *Manager) GetGame(gameID string) (*GameState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	g, ok := m.games[gameID]
	return g, ok
}

// GameForUser returns active game id for a username or fallback.
func (m *Manager) GameForUser(username, fallback string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if id, ok := m.userToGame[username]; ok {
		return id
	}
	return fallback
}

// GetGameByUser retrieves a game using username if present.
func (m *Manager) GetGameByUser(username string) (*GameState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if id, ok := m.userToGame[username]; ok {
		if g, exists := m.games[id]; exists {
			return g, true
		}
	}
	return nil, false
}

func (m *Manager) Abandon(username string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.userToGame, username)
	if m.waiting != nil && m.waiting.Username == username {
		m.waiting = nil
	}
}

// MarkDisconnected updates last seen time so sweeper can forfeit
// after the reconnect window.
func (m *Manager) MarkDisconnected(username string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id, ok := m.userToGame[username]; ok {
		if g, exists := m.games[id]; exists {
			g.LastMoveAt = time.Now()
		}
	}
}

// Forfeit stale games past reconnect window.
func (m *Manager) SweepDisconnects() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for id, g := range m.games {
		if g.Status != StatusFinished && now.Sub(g.LastMoveAt) > m.reconnectAfter {
			g.Status = StatusFinished
			g.Winner = findRemainingPlayer(g)
			g.EndedAt = now
			if m.onFinish != nil {
				go m.onFinish(g)
			}
			log.Printf("game %s forfeited due to timeout", id)
		}
	}
}

func findRemainingPlayer(g *GameState) string {
	for name, p := range g.Players {
		if p.IsBot {
			continue
		}
		return name
	}
	return "bot"
}

