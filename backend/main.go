package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

var gameServer *GameServer

func main() {
	// Get configuration from environment
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "fourinarow")
	
	kafkaBroker := getEnv("KAFKA_BROKER", "localhost:9092")
	kafkaTopic := getEnv("KAFKA_TOPIC", "game-events")
	
	port := getEnv("PORT", "8080")
	
	// Initialize database
	connStr := "host=" + dbHost + " port=" + dbPort + " user=" + dbUser + 
		" password=" + dbPassword + " dbname=" + dbName + " sslmode=require"
	
	db, err := NewDatabase(connStr)
	if err != nil {
		log.Printf("Warning: Database connection failed: %v", err)
		log.Println("Continuing without database...")
		db = nil
	} else {
		log.Println("Database connected successfully")
		defer db.Close()
	}
	
	// Initialize Kafka producer (optional)
	var kafka *KafkaProducer
	if kafkaBroker != "" {
		kafka = NewKafkaProducer([]string{kafkaBroker}, kafkaTopic)
		if kafka != nil {
			defer kafka.Close()
			log.Println("Kafka producer initialized")
		}
	} else {
		log.Println("Kafka not configured - analytics disabled")
	}
	
	// Initialize game server
	gameServer = NewGameServer(db, kafka)
	log.Println("Game server initialized")
	
	// HTTP handlers
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/api/leaderboard", handleLeaderboard)
	http.HandleFunc("/api/health", handleHealth)
	http.HandleFunc("/api/metrics", handleMetrics)
	
	// CORS middleware
	handler := enableCORS(http.DefaultServeMux)
	
	// Create server with timeouts
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	log.Printf("Server starting on port %s...", port)
	
	// Graceful shutdown handling
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		
		log.Println("Shutdown signal received, gracefully shutting down...")
		
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		log.Println("Server stopped")
	}()
	
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server error:", err)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	
	gameServer.HandleConnection(conn)
}

func handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	if gameServer.database == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}
	
	leaderboard, err := gameServer.database.GetLeaderboard(10)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(leaderboard)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	dbStatus := "disconnected"
	if gameServer.database != nil {
		dbStatus = "connected"
	}
	
	kafkaStatus := "disabled"
	if gameServer.kafka != nil {
		kafkaStatus = "connected"
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"database": dbStatus,
		"kafka":    kafkaStatus,
		"timestamp": time.Now().Unix(),
	})
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	if gameServer == nil {
		http.Error(w, "Server not initialized", http.StatusServiceUnavailable)
		return
	}
	
	gameServer.mu.RLock()
	defer gameServer.mu.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"activeGames":    len(gameServer.games),
		"waitingPlayers": len(gameServer.waitingPlayers),
		"totalPlayers":   len(gameServer.playerGames),
	})
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
