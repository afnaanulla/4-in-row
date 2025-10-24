package main

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

const (
	Rows    = 6
	Cols    = 7
	Empty   = 0
	Player1 = 1
	Player2 = 2
)

type Game struct {
	ID              string
	Board           [Rows][Cols]int
	Player1         *Player
	Player2         *Player
	CurrentTurn     int
	Status          string // "waiting", "playing", "finished"
	Winner          int
	StartTime       time.Time
	EndTime         time.Time
	MoveCount       int
	LastActivityTime time.Time
}

type Player struct {
	Username   string
	PlayerNum  int
	Conn       *websocket.Conn
	IsBot      bool
	Connected  bool
	LastSeen   time.Time
}

func NewGame(gameID string) *Game {
	return &Game{
		ID:              gameID,
		Board:           [Rows][Cols]int{},
		CurrentTurn:     Player1,
		Status:          "waiting",
		StartTime:       time.Now(),
		LastActivityTime: time.Now(),
	}
}

func (g *Game) MakeMove(col int, playerNum int) error {
	// Comprehensive validation
	if g == nil {
		return fmt.Errorf("game is nil")
	}
	
	if g.Status != "playing" {
		return fmt.Errorf("game not in playing state: %s", g.Status)
	}
	
	if g.CurrentTurn != playerNum {
		return fmt.Errorf("not your turn (current: %d, yours: %d)", g.CurrentTurn, playerNum)
	}
	
	if col < 0 || col >= Cols {
		return fmt.Errorf("invalid column %d (must be 0-%d)", col, Cols-1)
	}
	
	if playerNum != Player1 && playerNum != Player2 {
		return fmt.Errorf("invalid player number: %d", playerNum)
	}
	
	// Find lowest available row
	row := -1
	for r := Rows - 1; r >= 0; r-- {
		if g.Board[r][col] == Empty {
			row = r
			break
		}
	}
	
	if row == -1 {
		return fmt.Errorf("column is full")
	}
	
	// Place the piece
	g.Board[row][col] = playerNum
	g.MoveCount++
	g.LastActivityTime = time.Now()
	
	// Check for win
	if g.CheckWin(row, col, playerNum) {
		g.Status = "finished"
		g.Winner = playerNum
		g.EndTime = time.Now()
		return nil
	}
	
	// Check for draw
	if g.IsBoardFull() {
		g.Status = "finished"
		g.Winner = 0
		g.EndTime = time.Now()
		return nil
	}
	
	// Switch turn
	if g.CurrentTurn == Player1 {
		g.CurrentTurn = Player2
	} else {
		g.CurrentTurn = Player1
	}
	
	return nil
}

func (g *Game) CheckWin(row, col, player int) bool {
	// Validate input
	if row < 0 || row >= Rows || col < 0 || col >= Cols {
		return false
	}
	
	if player != Player1 && player != Player2 {
		return false
	}
	
	// Check horizontal
	if g.checkDirection(row, col, player, 0, 1) {
		return true
	}
	// Check vertical
	if g.checkDirection(row, col, player, 1, 0) {
		return true
	}
	// Check diagonal (top-left to bottom-right)
	if g.checkDirection(row, col, player, 1, 1) {
		return true
	}
	// Check diagonal (bottom-left to top-right)
	if g.checkDirection(row, col, player, 1, -1) {
		return true
	}
	return false
}

func (g *Game) checkDirection(row, col, player, dRow, dCol int) bool {
	count := 1
	
	// Check positive direction
	r, c := row+dRow, col+dCol
	for r >= 0 && r < Rows && c >= 0 && c < Cols && g.Board[r][c] == player {
		count++
		r += dRow
		c += dCol
	}
	
	// Check negative direction
	r, c = row-dRow, col-dCol
	for r >= 0 && r < Rows && c >= 0 && c < Cols && g.Board[r][c] == player {
		count++
		r -= dRow
		c -= dCol
	}
	
	return count >= 4
}

func (g *Game) IsBoardFull() bool {
	for c := 0; c < Cols; c++ {
		if g.Board[0][c] == Empty {
			return false
		}
	}
	return true
}

func (g *Game) GetValidMoves() []int {
	validMoves := []int{}
	for c := 0; c < Cols; c++ {
		if g.Board[0][c] == Empty {
			validMoves = append(validMoves, c)
		}
	}
	return validMoves
}

func (g *Game) SimulateMove(col, player int) (int, int, bool) {
	// Validate inputs
	if col < 0 || col >= Cols {
		return -1, -1, false
	}
	
	if player != Player1 && player != Player2 {
		return -1, -1, false
	}
	
	// Find the row where piece would land
	row := -1
	for r := Rows - 1; r >= 0; r-- {
		if g.Board[r][col] == Empty {
			row = r
			break
		}
	}
	
	if row == -1 {
		return -1, -1, false
	}
	
	// Temporarily place piece
	g.Board[row][col] = player
	wins := g.CheckWin(row, col, player)
	// Remove piece
	g.Board[row][col] = Empty
	
	return row, col, wins
}
