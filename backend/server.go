package main

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type GameServer struct {
	games          map[string]*Game
	waitingPlayers []*Player
	playerGames    map[string]string // username -> gameID
	mu             sync.RWMutex
	database       *Database
	kafka          *KafkaProducer
}

func NewGameServer(db *Database, kafka *KafkaProducer) *GameServer {
	gs := &GameServer{
		games:       make(map[string]*Game),
		playerGames: make(map[string]string),
		database:    db,
		kafka:       kafka,
	}
	
	// Start background tasks
	go gs.matchmakingLoop()
	go gs.cleanupLoop()
	
	return gs
}

func (gs *GameServer) HandleConnection(conn *websocket.Conn) {
	defer conn.Close()
	
	var username string
	
	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("Error reading message: %v", err)
			if username != "" {
				gs.handleDisconnect(username)
			}
			break
		}
		
		switch msg.Type {
		case "join":
			username = msg.Username
			gs.handleJoin(conn, username)
		case "move":
			gs.handleMoveRequest(username, msg.Column)
		case "reconnect":
			username = msg.Username
			gs.handleReconnect(conn, username)
		}
	}
}

func (gs *GameServer) handleJoin(conn *websocket.Conn, username string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	
	// Validate username
	if username == "" || len(username) > 50 {
		conn.WriteJSON(Message{
			Type: "error",
			Data: map[string]interface{}{"message": "Invalid username. Must be 1-50 characters."},
		})
		return
	}
	
	// Sanitize username (basic validation)
	if !isValidUsername(username) {
		conn.WriteJSON(Message{
			Type: "error",
			Data: map[string]interface{}{"message": "Invalid username. Only alphanumeric and basic characters allowed."},
		})
		return
	}
	
	// Check if player is already in a game
	if gameID, exists := gs.playerGames[username]; exists {
		game := gs.games[gameID]
		if game != nil && game.Status == "playing" {
			// Reconnect to existing game
			gs.reconnectPlayer(game, username, conn)
			return
		}
		// Clean up stale entry
		delete(gs.playerGames, username)
	}
	
	// Check if already waiting
	for _, wp := range gs.waitingPlayers {
		if wp.Username == username {
			log.Printf("Player %s already waiting", username)
			return
		}
	}
	
	player := &Player{
		Username:  username,
		Conn:      conn,
		Connected: true,
		LastSeen:  time.Now(),
	}
	
	gs.waitingPlayers = append(gs.waitingPlayers, player)
	
	// Send waiting message
	if err := conn.WriteJSON(Message{
		Type: "waiting",
		Data: map[string]interface{}{
			"message": "Waiting for opponent...",
		},
	}); err != nil {
		log.Printf("Error sending waiting message: %v", err)
	}
}

func (gs *GameServer) matchmakingLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		gs.mu.Lock()
		
		if len(gs.waitingPlayers) >= 2 {
			// Match two players
			p1 := gs.waitingPlayers[0]
			p2 := gs.waitingPlayers[1]
			gs.waitingPlayers = gs.waitingPlayers[2:]
			
			gs.mu.Unlock()
			gs.createGame(p1, p2, false)
			gs.mu.Lock()
		} else if len(gs.waitingPlayers) == 1 {
			// Check if player has been waiting for 10 seconds
			player := gs.waitingPlayers[0]
			if time.Since(player.LastSeen) > 10*time.Second {
				gs.waitingPlayers = gs.waitingPlayers[1:]
				gs.mu.Unlock()
				
				// Create bot player
				botPlayer := &Player{
					Username:  "BOT",
					IsBot:     true,
					Connected: true,
				}
				gs.createGame(player, botPlayer, true)
				gs.mu.Lock()
			}
		}
		
		gs.mu.Unlock()
	}
}

func (gs *GameServer) createGame(p1, p2 *Player, withBot bool) {
	gameID := uuid.New().String()
	game := NewGame(gameID)
	
	p1.PlayerNum = Player1
	p2.PlayerNum = Player2
	
	game.Player1 = p1
	game.Player2 = p2
	game.Status = "playing"
	game.StartTime = time.Now()
	
	gs.mu.Lock()
	gs.games[gameID] = game
	gs.playerGames[p1.Username] = gameID
	if !withBot {
		gs.playerGames[p2.Username] = gameID
	}
	gs.mu.Unlock()
	
	// Send game start messages
	gameState := gs.getGameState(game)
	
	if p1.Conn != nil {
		p1.Conn.WriteJSON(Message{
			Type: "game_start",
			Data: map[string]interface{}{
				"gameId":      gameID,
				"playerNum":   Player1,
				"opponent":    p2.Username,
				"opponentIsBot": withBot,
				"gameState":   gameState,
			},
		})
	}
	
	if !withBot && p2.Conn != nil {
		p2.Conn.WriteJSON(Message{
			Type: "game_start",
			Data: map[string]interface{}{
				"gameId":      gameID,
				"playerNum":   Player2,
				"opponent":    p1.Username,
				"opponentIsBot": false,
				"gameState":   gameState,
			},
		})
	}
	
	// Send Kafka event
	if gs.kafka != nil {
		gs.kafka.SendEvent(GameEvent{
			EventType:    "game_start",
			GameID:       gameID,
			Timestamp:    time.Now(),
			Player1:      p1.Username,
			Player2:      p2.Username,
			Player2IsBot: withBot,
		})
	}
	
	// If playing with bot, bot makes first move if it's bot's turn
	if withBot && game.CurrentTurn == Player2 {
		go func() {
			bot := NewBot(Player2)
			bot.MakeMoveWithDelay(game, gs)
		}()
	}
}

func (gs *GameServer) handleMoveRequest(username string, col int) {
	gs.mu.RLock()
	gameID, exists := gs.playerGames[username]
	if !exists {
		gs.mu.RUnlock()
		return
	}
	game := gs.games[gameID]
	gs.mu.RUnlock()
	
	if game == nil {
		return
	}
	
	var playerNum int
	if game.Player1.Username == username {
		playerNum = Player1
	} else {
		playerNum = Player2
	}
	
	gs.handleMove(game, col, playerNum)
}

func (gs *GameServer) handleMove(game *Game, col int, playerNum int) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	
	// Validate game state
	if game == nil {
		log.Printf("Error: Game is nil")
		return
	}
	
	if game.Status != "playing" {
		log.Printf("Error: Game not in playing state, status: %s", game.Status)
		return
	}
	
	// Validate column range
	if col < 0 || col >= Cols {
		log.Printf("Error: Invalid column %d, must be 0-%d", col, Cols-1)
		return
	}
	
	err := game.MakeMove(col, playerNum)
	if err != nil {
		log.Printf("Invalid move: %v", err)
		return
	}
	
	// Send Kafka event
	if gs.kafka != nil {
		gs.kafka.SendEvent(GameEvent{
			EventType: "move",
			GameID:    game.ID,
			Timestamp: time.Now(),
			Player1:   game.Player1.Username,
			Player2:   game.Player2.Username,
			Player2IsBot: game.Player2.IsBot,
			Move: &MoveData{
				PlayerNum: playerNum,
				Column:    col,
				MoveNum:   game.MoveCount,
			},
		})
	}
	
	// Broadcast game state with winner info
	gameState := gs.getGameState(game)
	
	// Add winner username if game is finished
	if game.Status == "finished" {
		if game.Winner == Player1 {
			gameState["winnerName"] = game.Player1.Username
		} else if game.Winner == Player2 {
			gameState["winnerName"] = game.Player2.Username
		} else {
			gameState["winnerName"] = ""
		}
	}
	
	msg := Message{
		Type: "game_update",
		Data: map[string]interface{}{
			"gameState": gameState,
		},
	}
	
	// Safe WebSocket writes with error handling
	if game.Player1.Conn != nil && game.Player1.Connected {
		if err := game.Player1.Conn.WriteJSON(msg); err != nil {
			log.Printf("Error writing to Player1: %v", err)
			game.Player1.Connected = false
		}
	}
	if game.Player2.Conn != nil && game.Player2.Connected && !game.Player2.IsBot {
		if err := game.Player2.Conn.WriteJSON(msg); err != nil {
			log.Printf("Error writing to Player2: %v", err)
			game.Player2.Connected = false
		}
	}
	
	// Handle game end
	if game.Status == "finished" {
		gs.handleGameEnd(game)
	} else if game.Player2.IsBot && game.CurrentTurn == Player2 {
		// Bot's turn - ensure game is still valid
		if game.Status == "playing" {
			go func() {
				bot := NewBot(Player2)
				bot.MakeMoveWithDelay(game, gs)
			}()
		}
	}
}

func (gs *GameServer) handleGameEnd(game *Game) {
	// Save to database
	if gs.database != nil {
		gs.database.SaveGame(game)
		
		// Update player stats
		if game.Winner == 0 {
			// Draw - log for debugging
			log.Printf("Game ended in draw: %s vs %s", game.Player1.Username, game.Player2.Username)
			if !game.Player1.IsBot {
				err := gs.database.UpdatePlayerStats(game.Player1.Username, false, true)
				if err != nil {
					log.Printf("Error updating Player1 draw stats: %v", err)
				} else {
					log.Printf("Updated draw stats for Player1: %s", game.Player1.Username)
				}
			}
			if !game.Player2.IsBot {
				err := gs.database.UpdatePlayerStats(game.Player2.Username, false, true)
				if err != nil {
					log.Printf("Error updating Player2 draw stats: %v", err)
				} else {
					log.Printf("Updated draw stats for Player2: %s", game.Player2.Username)
				}
			}
		} else {
			// Someone won
			winner := game.Player1
			loser := game.Player2
			if game.Winner == Player2 {
				winner = game.Player2
				loser = game.Player1
			}
			
			if !winner.IsBot {
				gs.database.UpdatePlayerStats(winner.Username, true, false)
			}
			if !loser.IsBot {
				gs.database.UpdatePlayerStats(loser.Username, false, false)
			}
		}
	}
	
	// Send Kafka event
	if gs.kafka != nil {
		duration := int(game.EndTime.Sub(game.StartTime).Seconds())
		gs.kafka.SendEvent(GameEvent{
			EventType:    "game_end",
			GameID:       game.ID,
			Timestamp:    time.Now(),
			Player1:      game.Player1.Username,
			Player2:      game.Player2.Username,
			Player2IsBot: game.Player2.IsBot,
			Result: &GameResult{
				Winner:       game.Winner,
				TotalMoves:   game.MoveCount,
				DurationSecs: duration,
			},
		})
	}
	
	// Clean up
	delete(gs.playerGames, game.Player1.Username)
	if !game.Player2.IsBot {
		delete(gs.playerGames, game.Player2.Username)
	}
}

func (gs *GameServer) handleDisconnect(username string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	
	gameID, exists := gs.playerGames[username]
	if !exists {
		return
	}
	
	game := gs.games[gameID]
	if game == nil {
		return
	}
	
	// Mark player as disconnected
	if game.Player1.Username == username {
		game.Player1.Connected = false
		game.Player1.LastSeen = time.Now()
	} else if game.Player2.Username == username {
		game.Player2.Connected = false
		game.Player2.LastSeen = time.Now()
	}
}

func (gs *GameServer) handleReconnect(conn *websocket.Conn, username string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	
	gameID, exists := gs.playerGames[username]
	if !exists {
		conn.WriteJSON(Message{Type: "error", Data: map[string]interface{}{"message": "No active game found"}})
		return
	}
	
	game := gs.games[gameID]
	if game == nil {
		conn.WriteJSON(Message{Type: "error", Data: map[string]interface{}{"message": "Game not found"}})
		return
	}
	
	gs.reconnectPlayer(game, username, conn)
}

func (gs *GameServer) reconnectPlayer(game *Game, username string, conn *websocket.Conn) {
	var player *Player
	if game.Player1.Username == username {
		player = game.Player1
	} else if game.Player2.Username == username {
		player = game.Player2
	}
	
	if player == nil {
		return
	}
	
	player.Conn = conn
	player.Connected = true
	player.LastSeen = time.Now()
	
	// Send current game state
	gameState := gs.getGameState(game)
	conn.WriteJSON(Message{
		Type: "reconnected",
		Data: map[string]interface{}{
			"gameId":    game.ID,
			"playerNum": player.PlayerNum,
			"gameState": gameState,
		},
	})
}

func (gs *GameServer) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		gs.mu.Lock()
		
		for gameID, game := range gs.games {
			if game.Status != "playing" {
				continue
			}
			
			// Check for disconnected players beyond 30 seconds
			now := time.Now()
			
			if !game.Player1.Connected && now.Sub(game.Player1.LastSeen) > 30*time.Second {
				game.Status = "finished"
				game.Winner = Player2
				game.EndTime = now
				gs.handleGameEnd(game)
				delete(gs.games, gameID)
			} else if !game.Player2.Connected && now.Sub(game.Player2.LastSeen) > 30*time.Second {
				game.Status = "finished"
				game.Winner = Player1
				game.EndTime = now
				gs.handleGameEnd(game)
				delete(gs.games, gameID)
			}
		}
		
		gs.mu.Unlock()
	}
}

func (gs *GameServer) getGameState(game *Game) map[string]interface{} {
	return map[string]interface{}{
		"board":       game.Board,
		"currentTurn": game.CurrentTurn,
		"status":      game.Status,
		"winner":      game.Winner,
		"moveCount":   game.MoveCount,
	}
}

type Message struct {
	Type     string                 `json:"type"`
	Username string                 `json:"username,omitempty"`
	Column   int                    `json:"column,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

// Validate username contains only safe characters
func isValidUsername(username string) bool {
	if len(username) == 0 || len(username) > 50 {
		return false
	}
	for _, char := range username {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || 
			(char >= '0' && char <= '9') || char == '_' || char == '-' || char == ' ') {
			return false
		}
	}
	return true
}
