package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"emittr/backend/internal/analytics"
	"emittr/backend/internal/server"
	"emittr/backend/internal/storage"
)

func main() {
	// Check for PORT first (used by Render, Fly.io, Heroku, etc.)
	port := os.Getenv("PORT")
	var addr string
	if port != "" {
		addr = ":" + port
	} else {
		addr = getEnv("ADDR", ":8080")
	}
	botDelay := durationEnv("BOT_DELAY", 10*time.Second)
	reconnect := durationEnv("RECONNECT_WINDOW", 30*time.Second)

	var store storage.Store
	if dsn := os.Getenv("POSTGRES_URL"); dsn != "" {
		pg, err := storage.NewPostgresStore(context.Background(), dsn)
		if err != nil {
			log.Printf("postgres disabled: %v", err)
		} else {
			if err := pg.EnsureTables(context.Background()); err != nil {
				log.Printf("postgres ensure tables failed: %v", err)
			}
			store = pg
		}
	}

	var producer *analytics.Producer
	if brokers := os.Getenv("KAFKA_BROKERS"); brokers != "" {
		topic := getEnv("KAFKA_TOPIC", "game-events")
		producer = analytics.NewProducer([]string{brokers}, topic)
	}

	srv := server.New(server.Config{
		BotFallbackAfter: botDelay,
		ReconnectWindow:  reconnect,
		Store:            store,
		Analytics:        producer,
	})

	log.Printf("server listening on %s", addr)
	if err := srv.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return time.Duration(parsed) * time.Second
		}
	}
	return fallback
}

