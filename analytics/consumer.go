package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

type event struct {
	Event     string                 `json:"event"`
	Payload   map[string]any         `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
}

type metrics struct {
	winnerCounts      map[string]int
	gameDurations     []float64
	gamesPerDay       map[string]int
	gamesPerHour      map[string]int
	userGames         map[string]int
	userWins          map[string]int
	totalGames        int
	mu                sync.Mutex
}

func newMetrics() *metrics {
	return &metrics{
		winnerCounts:  make(map[string]int),
		gameDurations: make([]float64, 0),
		gamesPerDay:   make(map[string]int),
		gamesPerHour:  make(map[string]int),
		userGames:     make(map[string]int),
		userWins:      make(map[string]int),
	}
}

func (m *metrics) recordGameFinished(payload map[string]any, timestamp time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalGames++

	// Track winner
	if winner, ok := payload["winner"].(string); ok && winner != "" && winner != "bot" {
		m.winnerCounts[winner]++
		m.userWins[winner]++
	}

	// Track game duration
	if duration, ok := payload["duration"].(float64); ok {
		m.gameDurations = append(m.gameDurations, duration)
	}

	// Track games per day/hour
	dayKey := timestamp.Format("2006-01-02")
	hourKey := timestamp.Format("2006-01-02 15:00")
	m.gamesPerDay[dayKey]++
	m.gamesPerHour[hourKey]++

	// Track user-specific metrics
	if players, ok := payload["players"].([]any); ok {
		for _, p := range players {
			if username, ok := p.(string); ok && username != "bot" {
				m.userGames[username]++
			}
		}
	}
}

func (m *metrics) getAverageDuration() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.gameDurations) == 0 {
		return 0
	}
	sum := 0.0
	for _, d := range m.gameDurations {
		sum += d
	}
	return sum / float64(len(m.gameDurations))
}

func (m *metrics) printStats() {
	m.mu.Lock()
	defer m.mu.Unlock()

	avgDuration := 0.0
	if len(m.gameDurations) > 0 {
		sum := 0.0
		for _, d := range m.gameDurations {
			sum += d
		}
		avgDuration = sum / float64(len(m.gameDurations))
	}

	log.Printf("=== ANALYTICS SUMMARY ===")
	log.Printf("Total Games: %d", m.totalGames)
	log.Printf("Average Game Duration: %.2f seconds", avgDuration)
	log.Printf("Most Frequent Winners: %v", m.winnerCounts)
	log.Printf("Games Per Day (last 7 days): %v", m.gamesPerDay)
	log.Printf("Games Per Hour (last 24 hours): %v", m.gamesPerHour)
	log.Printf("User Game Counts: %v", m.userGames)
	log.Printf("User Win Counts: %v", m.userWins)
	log.Printf("========================")
}

func main() {
	broker := getenv("KAFKA_BROKER", "localhost:9092")
	topic := getenv("KAFKA_TOPIC", "game-events")

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{broker},
		Topic:   topic,
		GroupID: "analytics-consumer",
	})
	defer reader.Close()

	log.Printf("analytics consumer listening on %s topic=%s", broker, topic)

	metrics := newMetrics()
	start := time.Now()

	// Print stats every 30 seconds
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			metrics.printStats()
		}
	}()

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Fatalf("read error: %v", err)
		}
		var e event
		if err := json.Unmarshal(msg.Value, &e); err != nil {
			log.Printf("failed to unmarshal event: %v", err)
			continue
		}

		if e.Event == "game_finished" {
			metrics.recordGameFinished(e.Payload, e.Timestamp)
		}

		// Log every event
		log.Printf("event=%s gameId=%v winner=%v", e.Event,
			e.Payload["gameId"],
			e.Payload["winner"])
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

