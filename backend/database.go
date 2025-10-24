package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(connectionString string) (*Database, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	database := &Database{db: db}
	if err := database.createTables(); err != nil {
		return nil, err
	}

	return database, nil
}

func (d *Database) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS games (
			id VARCHAR(255) PRIMARY KEY,
			player1_username VARCHAR(255),
			player2_username VARCHAR(255),
			winner INTEGER,
			status VARCHAR(50),
			start_time TIMESTAMP,
			end_time TIMESTAMP,
			move_count INTEGER,
			duration_seconds INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS players (
			username VARCHAR(255) PRIMARY KEY,
			games_played INTEGER DEFAULT 0,
			games_won INTEGER DEFAULT 0,
			games_lost INTEGER DEFAULT 0,
			games_drawn INTEGER DEFAULT 0,
			total_moves INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_players_wins ON players(games_won DESC)`,
	}

	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

func (d *Database) SaveGame(game *Game) error {
	duration := 0
	if !game.EndTime.IsZero() {
		duration = int(game.EndTime.Sub(game.StartTime).Seconds())
	}

	player1Username := ""
	player2Username := ""
	
	if game.Player1 != nil {
		player1Username = game.Player1.Username
	}
	if game.Player2 != nil {
		player2Username = game.Player2.Username
	}

	_, err := d.db.Exec(`
		INSERT INTO games (id, player1_username, player2_username, winner, status, start_time, end_time, move_count, duration_seconds)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			winner = EXCLUDED.winner,
			status = EXCLUDED.status,
			end_time = EXCLUDED.end_time,
			move_count = EXCLUDED.move_count,
			duration_seconds = EXCLUDED.duration_seconds
	`, game.ID, player1Username, player2Username, game.Winner, game.Status, game.StartTime, game.EndTime, game.MoveCount, duration)

	return err
}

func (d *Database) UpdatePlayerStats(username string, won bool, drawn bool) error {
	// Ensure player exists
	_, err := d.db.Exec(`
		INSERT INTO players (username, games_played, games_won, games_lost, games_drawn)
		VALUES ($1, 0, 0, 0, 0)
		ON CONFLICT (username) DO NOTHING
	`, username)
	if err != nil {
		return err
	}

	// Update stats
	if drawn {
		_, err = d.db.Exec(`
			UPDATE players SET
				games_played = games_played + 1,
				games_drawn = games_drawn + 1
			WHERE username = $1
		`, username)
	} else if won {
		_, err = d.db.Exec(`
			UPDATE players SET
				games_played = games_played + 1,
				games_won = games_won + 1
			WHERE username = $1
		`, username)
	} else {
		_, err = d.db.Exec(`
			UPDATE players SET
				games_played = games_played + 1,
				games_lost = games_lost + 1
			WHERE username = $1
		`, username)
	}

	return err
}

func (d *Database) GetLeaderboard(limit int) ([]LeaderboardEntry, error) {
	rows, err := d.db.Query(`
		SELECT username, games_played, games_won, games_lost, games_drawn
		FROM players
		WHERE username != 'BOT' AND games_played > 0
		ORDER BY games_won DESC, (games_won::float / NULLIF(games_played, 0)) DESC, games_played DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leaderboard []LeaderboardEntry
	for rows.Next() {
		var entry LeaderboardEntry
		if err := rows.Scan(&entry.Username, &entry.GamesPlayed, &entry.GamesWon, &entry.GamesLost, &entry.GamesDrawn); err != nil {
			log.Printf("Error scanning leaderboard entry: %v", err)
			continue
		}
		leaderboard = append(leaderboard, entry)
	}

	return leaderboard, nil
}

func (d *Database) GetPlayerStats(username string) (*PlayerStats, error) {
	var stats PlayerStats
	err := d.db.QueryRow(`
		SELECT username, games_played, games_won, games_lost, games_drawn, total_moves, created_at
		FROM players
		WHERE username = $1
	`, username).Scan(&stats.Username, &stats.GamesPlayed, &stats.GamesWon, &stats.GamesLost, &stats.GamesDrawn, &stats.TotalMoves, &stats.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("player not found")
	}
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

type LeaderboardEntry struct {
	Username    string `json:"username"`
	GamesPlayed int    `json:"gamesPlayed"`
	GamesWon    int    `json:"gamesWon"`
	GamesLost   int    `json:"gamesLost"`
	GamesDrawn  int    `json:"gamesDrawn"`
}

type PlayerStats struct {
	Username    string    `json:"username"`
	GamesPlayed int       `json:"gamesPlayed"`
	GamesWon    int       `json:"gamesWon"`
	GamesLost   int       `json:"gamesLost"`
	GamesDrawn  int       `json:"gamesDrawn"`
	TotalMoves  int       `json:"totalMoves"`
	CreatedAt   time.Time `json:"createdAt"`
}
