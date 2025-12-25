package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"emittr/backend/internal/analytics"
	"emittr/backend/internal/game"
	"emittr/backend/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Server struct {
	router          *gin.Engine
	manager         *game.Manager
	store           storage.Store
	analytics       *analytics.Producer
	inMemoryWins    map[string]int
	winMu           sync.Mutex
	connections     map[string]*wsClient
	connMu          sync.RWMutex
	botDelay        time.Duration
	reconnectWindow time.Duration
}

type Config struct {
	BotFallbackAfter time.Duration
	ReconnectWindow  time.Duration
	Store            storage.Store
	Analytics        *analytics.Producer
}

func New(cfg Config) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	s := &Server{
		router:          router,
		manager:         game.NewManager(cfg.ReconnectWindow, nil),
		store:           cfg.Store,
		analytics:       cfg.Analytics,
		inMemoryWins:    make(map[string]int),
		connections:     make(map[string]*wsClient),
		botDelay:        cfg.BotFallbackAfter,
		reconnectWindow: cfg.ReconnectWindow,
	}
	s.manager = game.NewManager(cfg.ReconnectWindow, s.onFinish)

	router.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	router.GET("/leaderboard", s.handleLeaderboard)
	router.GET("/ws", s.handleWS)
	
	// Serve frontend static files
	frontendPath := filepath.Join("..", "frontend")
	router.StaticFile("/", filepath.Join(frontendPath, "index.html"))
	router.Static("/static", frontendPath)
	
	return s
}

func (s *Server) Run(addr string) error {
	go s.sweeper()
	return s.router.Run(addr)
}

func (s *Server) sweeper() {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		s.manager.SweepDisconnects()
	}
}

func (s *Server) handleLeaderboard(c *gin.Context) {
	ctx := c.Request.Context()
	if s.store != nil {
		rows, err := s.store.GetLeaderboard(ctx, 10)
		if err == nil {
			c.JSON(http.StatusOK, rows)
			return
		}
		log.Printf("leaderboard db error: %v", err)
	}
	// fallback in-memory
	type pair struct {
		Username string `json:"username"`
		Wins     int    `json:"wins"`
	}
	var res []pair
	s.winMu.Lock()
	for k, v := range s.inMemoryWins {
		res = append(res, pair{Username: k, Wins: v})
	}
	s.winMu.Unlock()
	c.JSON(http.StatusOK, res)
}

type wsClient struct {
	username string
	conn     *websocket.Conn
	send     chan []byte
	server   *Server
	gameID   string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func (s *Server) handleWS(c *gin.Context) {
	username := c.Query("username")
	requestGameID := c.Query("gameId")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := &wsClient{
		username: username,
		conn:     conn,
		send:     make(chan []byte, 8),
		server:   s,
		gameID:   requestGameID,
	}
	s.register(client)

	go client.writePump()
	go client.readPump()
}

func (s *Server) register(c *wsClient) {
	s.connMu.Lock()
	s.connections[c.username] = c
	s.connMu.Unlock()
}

func (s *Server) unregister(c *wsClient) {
	s.connMu.Lock()
	delete(s.connections, c.username)
	s.connMu.Unlock()
	c.conn.Close()
}

func (c *wsClient) writePump() {
	for msg := range c.send {
		_ = c.conn.WriteMessage(websocket.TextMessage, msg)
	}
}

func (c *wsClient) readPump() {
	defer c.server.unregister(c)
	s := c.server

	var gameState *game.GameState

	// Rejoin if gameId provided
	if c.gameID != "" {
		if g, ok := s.manager.GetGame(c.gameID); ok {
			if _, exists := g.Players[c.username]; exists {
				gameState = g
				s.pushInit(g, c.username)
				s.pushState(g)
			}
		}
	}
	if gameState == nil {
		g, _, waiting := s.manager.AssignPlayer(c.username)
		if waiting {
			c.sendJSON(map[string]any{"type": "waiting", "message": "waiting for opponent"})
			time.AfterFunc(s.botDelay, func() {
				// Only trigger if still unpaired
				if _, ok := s.manager.GetGameByUser(c.username); !ok {
					g := s.manager.StartBotGame(c.username)
					s.pushInit(g, c.username)
					if g.Bot != nil && g.Turn == game.CellP2 {
						s.playBotTurn(g)
					}
				}
			})
		} else {
			gameState = g
			c.server.pushInit(gameState, c.username)
			// notify opponent if online
			for uname, pl := range gameState.Players {
				if uname == c.username {
					continue
				}
				if pl.IsBot {
					continue
				}
				if peer, ok := s.connections[uname]; ok {
					s.pushInit(gameState, peer.username)
				}
			}
		}
	} else {
		s.pushInit(gameState, c.username)
	}

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			s.manager.MarkDisconnected(c.username)
			return
		}
		var msg map[string]any
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg["type"] == "move" {
			col, ok := msg["column"].(float64)
			if !ok {
				continue
			}
			move := game.Move{
				Username: c.username,
				GameID:   s.manager.GameForUser(c.username, c.gameID),
				Column:   int(col),
			}
			res, g, err := s.manager.HandleMove(move)
			if err != nil {
				c.sendJSON(map[string]any{"type": "error", "message": err.Error()})
				continue
			}
			s.broadcastState(g, res)
			if g.Bot != nil && g.Status == game.StatusActive && g.Turn == g.Players["bot"].Slot {
				s.playBotTurn(g)
			}
		}
	}
}

func (s *Server) pushInit(g *game.GameState, username string) {
	slot := 0
	if p, ok := g.Players[username]; ok {
		slot = p.Slot
	}
	payload := map[string]any{
		"type":      "init",
		"gameId":    g.ID,
		"board":     g.Board,
		"turn":      g.Turn,
		"you":       username,
		"slot":      slot,
		"opponent":  s.findOpponent(g, username),
		"status":    g.Status,
		"winner":    g.Winner,
		"timestamp": time.Now().UTC(),
	}
	s.sendToUser(username, payload)
}

func (s *Server) pushState(g *game.GameState) {
	res := game.MoveResult{Board: g.Board}
	s.broadcastState(g, res)
}

func (s *Server) broadcastState(g *game.GameState, res game.MoveResult) {
	payload := map[string]any{
		"type":   "state",
		"board":  res.Board,
		"turn":   g.Turn,
		"status": g.Status,
		"winner": g.Winner,
	}
	for uname := range g.Players {
		if uname == "bot" {
			continue
		}
		s.sendToUser(uname, payload)
	}
	if s.analytics != nil {
		players := make([]string, 0, len(g.Players))
		for uname := range g.Players {
			if uname != "bot" {
				players = append(players, uname)
			}
		}
		s.analytics.Publish(context.Background(), "move_played", map[string]any{
			"gameId":  g.ID,
			"status":  g.Status,
			"winner":  g.Winner,
			"players": players,
		})
	}
}

func (s *Server) sendToUser(username string, payload map[string]any) {
	s.connMu.RLock()
	client, ok := s.connections[username]
	s.connMu.RUnlock()
	if !ok {
		return
	}
	data, _ := json.Marshal(payload)
	select {
	case client.send <- data:
	default:
	}
}

func (s *Server) findOpponent(g *game.GameState, username string) string {
	for name, p := range g.Players {
		if name != username && !p.IsBot {
			return name
		}
	}
	if g.Bot != nil {
		return "bot"
	}
	return ""
}

func (s *Server) onFinish(g *game.GameState) {
	if g.Winner != "" && g.Winner != "bot" {
		s.winMu.Lock()
		s.inMemoryWins[g.Winner]++
		s.winMu.Unlock()
	}
	if s.store != nil {
		_ = s.store.SaveGame(context.Background(), storage.CompletedGame{
			ID:        g.ID,
			Winner:    g.Winner,
			Status:    g.Status,
			StartedAt: g.StartedAt,
			EndedAt:   g.EndedAt,
		})
	}
	if s.analytics != nil {
		players := make([]string, 0, len(g.Players))
		for uname := range g.Players {
			if uname != "bot" {
				players = append(players, uname)
			} else {
				players = append(players, "bot")
			}
		}
		duration := g.EndedAt.Sub(g.StartedAt).Seconds()
		s.analytics.Publish(context.Background(), "game_finished", map[string]any{
			"gameId":   g.ID,
			"winner":   g.Winner,
			"status":   g.Status,
			"players":  players,
			"duration": duration,
			"startedAt": g.StartedAt,
			"endedAt":   g.EndedAt,
		})
	}
}

func (s *Server) playBotTurn(g *game.GameState) {
	bot := g.Bot
	if bot == nil {
		return
	}
	col := bot.ChooseMove(g.Board)
	move := game.Move{
		Username: "bot",
		GameID:   g.ID,
		Column:   col,
	}
	res, g, err := s.manager.HandleMove(move)
	if err != nil {
		return
	}
	s.broadcastState(g, res)
}

func (c *wsClient) sendJSON(v any) {
	data, _ := json.Marshal(v)
	select {
	case c.send <- data:
	default:
	}
}

