package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/segmentio/kafka-go"
	_ "github.com/lib/pq"
)

type GameEvent struct {
	EventType    string       `json:"eventType"`
	GameID       string       `json:"gameId"`
	Timestamp    time.Time    `json:"timestamp"`
	Player1      string       `json:"player1"`
	Player2      string       `json:"player2"`
	Player2IsBot bool         `json:"player2IsBot"`
	Move         *MoveData    `json:"move,omitempty"`
	Result       *GameResult  `json:"result,omitempty"`
}

type MoveData struct {
	PlayerNum int `json:"playerNum"`
	Column    int `json:"column"`
	MoveNum   int `json:"moveNum"`
}

type GameResult struct {
	Winner       int `json:"winner"`
	TotalMoves   int `json:"totalMoves"`
	DurationSecs int `json:"durationSecs"`
}

type Analytics struct {
	db *sql.DB
}

func NewAnalytics(connStr string) (*Analytics, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	analytics := &Analytics{db: db}
	if err := analytics.createTables(); err != nil {
		return nil, err
	}

	return analytics, nil
}

func (a *Analytics) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS analytics_events (
			id SERIAL PRIMARY KEY,
			event_type VARCHAR(50),
			game_id VARCHAR(255),
			timestamp TIMESTAMP,
			player1 VARCHAR(255),
			player2 VARCHAR(255),
			player2_is_bot BOOLEAN,
			data JSONB
		)`,
		`CREATE TABLE IF NOT EXISTS analytics_metrics (
			id SERIAL PRIMARY KEY,
			metric_name VARCHAR(100),
			metric_value NUMERIC,
			calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_events_type ON analytics_events(event_type)`,
		`CREATE INDEX IF NOT EXISTS idx_events_timestamp ON analytics_events(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_events_game ON analytics_events(game_id)`,
	}

	for _, query := range queries {
		if _, err := a.db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

func (a *Analytics) ProcessEvent(event GameEvent) {
	// Store raw event
	data, _ := json.Marshal(event)
	
	_, err := a.db.Exec(`
		INSERT INTO analytics_events (event_type, game_id, timestamp, player1, player2, player2_is_bot, data)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, event.EventType, event.GameID, event.Timestamp, event.Player1, event.Player2, event.Player2IsBot, data)
	
	if err != nil {
		log.Printf("Error storing event: %v", err)
		return
	}

	// Process specific event types
	switch event.EventType {
	case "game_start":
		log.Printf("Game started: %s (%s vs %s)", event.GameID, event.Player1, event.Player2)
	case "move":
		if event.Move != nil {
			log.Printf("Move in game %s: Player %d -> Column %d", event.GameID, event.Move.PlayerNum, event.Move.Column)
		}
	case "game_end":
		if event.Result != nil {
			log.Printf("Game ended: %s - Winner: %d, Moves: %d, Duration: %ds", 
				event.GameID, event.Result.Winner, event.Result.TotalMoves, event.Result.DurationSecs)
			a.updateMetrics()
		}
	}
}

func (a *Analytics) updateMetrics() {
	// Calculate average game duration
	var avgDuration float64
	err := a.db.QueryRow(`
		SELECT AVG((data->>'result'->>'durationSecs')::numeric)
		FROM analytics_events
		WHERE event_type = 'game_end'
		AND timestamp > NOW() - INTERVAL '24 hours'
	`).Scan(&avgDuration)
	
	if err == nil {
		a.db.Exec(`
			INSERT INTO analytics_metrics (metric_name, metric_value)
			VALUES ('avg_game_duration_24h', $1)
		`, avgDuration)
	}

	// Count games in last hour
	var gamesLastHour int
	a.db.QueryRow(`
		SELECT COUNT(*)
		FROM analytics_events
		WHERE event_type = 'game_start'
		AND timestamp > NOW() - INTERVAL '1 hour'
	`).Scan(&gamesLastHour)
	
	a.db.Exec(`
		INSERT INTO analytics_metrics (metric_name, metric_value)
		VALUES ('games_per_hour', $1)
	`, gamesLastHour)

	// Count games per day
	var gamesToday int
	a.db.QueryRow(`
		SELECT COUNT(*)
		FROM analytics_events
		WHERE event_type = 'game_start'
		AND DATE(timestamp) = CURRENT_DATE
	`).Scan(&gamesToday)
	
	a.db.Exec(`
		INSERT INTO analytics_metrics (metric_name, metric_value)
		VALUES ('games_today', $1)
	`, gamesToday)
}

func (a *Analytics) Close() error {
	return a.db.Close()
}

func main() {
	// Configuration
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "fourinarow")
	
	kafkaBroker := getEnv("KAFKA_BROKER", "localhost:9092")
	kafkaTopic := getEnv("KAFKA_TOPIC", "game-events")
	kafkaGroup := getEnv("KAFKA_GROUP", "analytics-consumer")

	// Initialize analytics
	connStr := "host=" + dbHost + " port=" + dbPort + " user=" + dbUser + 
		" password=" + dbPassword + " dbname=" + dbName + " sslmode=disable"
	
	analytics, err := NewAnalytics(connStr)
	if err != nil {
		log.Fatal("Failed to initialize analytics:", err)
	}
	defer analytics.Close()
	
	log.Println("Analytics service connected to database")

	// Create Kafka reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{kafkaBroker},
		Topic:    kafkaTopic,
		GroupID:  kafkaGroup,
		MinBytes: 10e1,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	log.Printf("Kafka consumer started (broker: %s, topic: %s)", kafkaBroker, kafkaTopic)

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)

	go func() {
		<-sigchan
		log.Println("Shutting down...")
		cancel()
	}()

	// Consume messages
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("Error reading message: %v", err)
				continue
			}

			var event GameEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("Error unmarshaling event: %v", err)
				continue
			}

			analytics.ProcessEvent(event)
		}
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
