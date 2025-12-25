package storage

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

type CompletedGame struct {
	ID        string
	Winner    string
	Status    string
	StartedAt time.Time
	EndedAt   time.Time
}

type LeaderboardRow struct {
	Username string `json:"username"`
	Wins     int    `json:"wins"`
}

type Store interface {
	SaveGame(ctx context.Context, game CompletedGame) error
	GetLeaderboard(ctx context.Context, limit int) ([]LeaderboardRow, error)
}

type PostgresStore struct {
	pool *pgx.Conn
}

func NewPostgresStore(ctx context.Context, url string) (*PostgresStore, error) {
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return nil, err
	}
	return &PostgresStore{pool: conn}, nil
}

func (p *PostgresStore) Close(ctx context.Context) {
	if p.pool != nil {
		_ = p.pool.Close(ctx)
	}
}

func (p *PostgresStore) EnsureTables(ctx context.Context) error {
	_, err := p.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS games (
	id TEXT PRIMARY KEY,
	winner TEXT,
	status TEXT,
	started_at TIMESTAMP,
	ended_at TIMESTAMP
);
`)
	return err
}

func (p *PostgresStore) SaveGame(ctx context.Context, game CompletedGame) error {
	if p == nil || p.pool == nil {
		return nil
	}
	_, err := p.pool.Exec(ctx, `INSERT INTO games (id, winner, status, started_at, ended_at)
VALUES ($1,$2,$3,$4,$5) ON CONFLICT (id) DO NOTHING`, game.ID, game.Winner, game.Status, game.StartedAt, game.EndedAt)
	if err != nil {
		log.Printf("failed to save game: %v", err)
	}
	return err
}

func (p *PostgresStore) GetLeaderboard(ctx context.Context, limit int) ([]LeaderboardRow, error) {
	rows, err := p.pool.Query(ctx, `
SELECT winner, COUNT(*) as wins
FROM games
WHERE winner IS NOT NULL AND winner <> ''
GROUP BY winner
ORDER BY wins DESC
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []LeaderboardRow
	for rows.Next() {
		var row LeaderboardRow
		if err := rows.Scan(&row.Username, &row.Wins); err != nil {
			return nil, err
		}
		res = append(res, row)
	}
	return res, rows.Err()
}

