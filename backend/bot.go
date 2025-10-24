package main

import (
	"math"
	"math/rand"
	"time"
)

type Bot struct {
	PlayerNum int
}

func NewBot(playerNum int) *Bot {
	return &Bot{
		PlayerNum: playerNum,
	}
}

func (b *Bot) GetBestMove(game *Game) int {
	// Strategy priority:
	// 1. Win immediately if possible
	// 2. Block opponent's immediate win
	// 3. Use minimax with alpha-beta pruning for optimal play
	
	opponent := Player1
	if b.PlayerNum == Player1 {
		opponent = Player2
	}
	
	validMoves := game.GetValidMoves()
	if len(validMoves) == 0 {
		return -1
	}
	
	// 1. Check if bot can win immediately
	for _, col := range validMoves {
		_, _, wins := game.SimulateMove(col, b.PlayerNum)
		if wins {
			return col
		}
	}
	
	// 2. Check if need to block opponent's immediate win
	for _, col := range validMoves {
		_, _, wins := game.SimulateMove(col, opponent)
		if wins {
			return col
		}
	}
	
	// 3. Use minimax with alpha-beta pruning for best strategic move
	bestScore := math.MinInt32
	bestMove := validMoves[len(validMoves)/2] // Default to center if all else fails
	alpha := math.MinInt32
	beta := math.MaxInt32
	
	for _, col := range validMoves {
		row := b.getLowestRow(game, col)
		if row == -1 {
			continue
		}
		
		// Make move
		game.Board[row][col] = b.PlayerNum
		
		// Calculate score using minimax
		score := b.minimax(game, 5, false, alpha, beta, opponent)
		
		// Undo move
		game.Board[row][col] = Empty
		
		if score > bestScore {
			bestScore = score
			bestMove = col
		}
		
		alpha = max(alpha, bestScore)
	}
	
	return bestMove
}

func (b *Bot) evaluateMove(game *Game, col int) int {
	score := 0
	
	// Find where piece would land
	row := -1
	for r := Rows - 1; r >= 0; r-- {
		if game.Board[r][col] == Empty {
			row = r
			break
		}
	}
	
	if row == -1 {
		return -1000
	}
	
	// Prefer center columns
	centerDistance := abs(col - Cols/2)
	score += (Cols - centerDistance) * 10
	
	// Place piece temporarily
	game.Board[row][col] = b.PlayerNum
	
	// Count potential threats in all directions
	score += b.countThreats(game, row, col) * 50
	
	// Remove piece
	game.Board[row][col] = Empty
	
	return score
}

func (b *Bot) countThreats(game *Game, row, col int) int {
	threats := 0
	
	// Check all four directions for potential 4-in-a-row
	directions := [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}
	
	for _, dir := range directions {
		count := 1
		empty := 0
		
		// Check positive direction
		r, c := row+dir[0], col+dir[1]
		for i := 0; i < 3; i++ {
			if r >= 0 && r < Rows && c >= 0 && c < Cols {
				if game.Board[r][c] == b.PlayerNum {
					count++
				} else if game.Board[r][c] == Empty {
					empty++
				} else {
					break
				}
			}
			r += dir[0]
			c += dir[1]
		}
		
		// Check negative direction
		r, c = row-dir[0], col-dir[1]
		for i := 0; i < 3; i++ {
			if r >= 0 && r < Rows && c >= 0 && c < Cols {
				if game.Board[r][c] == b.PlayerNum {
					count++
				} else if game.Board[r][c] == Empty {
					empty++
				} else {
					break
				}
			}
			r -= dir[0]
			c -= dir[1]
		}
		
		// If we have 2-3 pieces with empty spaces, it's a threat
		if count >= 2 && empty >= 1 {
			threats++
		}
	}
	
	return threats
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Minimax algorithm with alpha-beta pruning
func (b *Bot) minimax(game *Game, depth int, isMaximizing bool, alpha, beta int, opponent int) int {
	// Check terminal states
	if depth == 0 {
		return b.evaluateBoard(game)
	}
	
	if game.IsBoardFull() {
		return 0 // Draw
	}
	
	validMoves := game.GetValidMoves()
	if len(validMoves) == 0 {
		return 0
	}
	
	if isMaximizing {
		// Bot's turn (maximize)
		maxScore := math.MinInt32
		
		for _, col := range validMoves {
			row := b.getLowestRow(game, col)
			if row == -1 {
				continue
			}
			
			game.Board[row][col] = b.PlayerNum
			
			// Check if this move wins
			if game.CheckWin(row, col, b.PlayerNum) {
				game.Board[row][col] = Empty
				return 10000 - (5 - depth) // Prefer faster wins
			}
			
			score := b.minimax(game, depth-1, false, alpha, beta, opponent)
			game.Board[row][col] = Empty
			
			maxScore = max(maxScore, score)
			alpha = max(alpha, score)
			
			if beta <= alpha {
				break // Beta cutoff
			}
		}
		
		return maxScore
	} else {
		// Opponent's turn (minimize)
		minScore := math.MaxInt32
		
		for _, col := range validMoves {
			row := b.getLowestRow(game, col)
			if row == -1 {
				continue
			}
			
			game.Board[row][col] = opponent
			
			// Check if opponent wins
			if game.CheckWin(row, col, opponent) {
				game.Board[row][col] = Empty
				return -10000 + (5 - depth) // Prefer blocking later losses
			}
			
			score := b.minimax(game, depth-1, true, alpha, beta, opponent)
			game.Board[row][col] = Empty
			
			minScore = min(minScore, score)
			beta = min(beta, score)
			
			if beta <= alpha {
				break // Alpha cutoff
			}
		}
		
		return minScore
	}
}

// Evaluate board position heuristically
func (b *Bot) evaluateBoard(game *Game) int {
	opponent := Player1
	if b.PlayerNum == Player1 {
		opponent = Player2
	}
	
	score := 0
	
	// Evaluate all positions
	for row := 0; row < Rows; row++ {
		for col := 0; col < Cols; col++ {
			if game.Board[row][col] == b.PlayerNum {
				score += b.evaluatePosition(game, row, col, b.PlayerNum)
			} else if game.Board[row][col] == opponent {
				score -= b.evaluatePosition(game, row, col, opponent)
			}
		}
	}
	
	// Bonus for center control
	centerCol := Cols / 2
	for row := 0; row < Rows; row++ {
		if game.Board[row][centerCol] == b.PlayerNum {
			score += 3
		}
	}
	
	return score
}

// Evaluate a single position's value
func (b *Bot) evaluatePosition(game *Game, row, col, player int) int {
	score := 0
	
	// Check all four directions
	directions := [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}
	
	for _, dir := range directions {
		count := 1
		openEnds := 0
		
		// Check positive direction
		r, c := row+dir[0], col+dir[1]
		for i := 0; i < 3; i++ {
			if r >= 0 && r < Rows && c >= 0 && c < Cols {
				if game.Board[r][c] == player {
					count++
				} else if game.Board[r][c] == Empty {
					openEnds++
					break
				} else {
					break
				}
			}
			r += dir[0]
			c += dir[1]
		}
		
		// Check negative direction
		r, c = row-dir[0], col-dir[1]
		for i := 0; i < 3; i++ {
			if r >= 0 && r < Rows && c >= 0 && c < Cols {
				if game.Board[r][c] == player {
					count++
				} else if game.Board[r][c] == Empty {
					openEnds++
					break
				} else {
					break
				}
			}
			r -= dir[0]
			c -= dir[1]
		}
		
		// Score based on count and open ends
		if count >= 4 {
			score += 1000 // Winning position
		} else if count == 3 && openEnds > 0 {
			score += 100 // Strong threat
		} else if count == 2 && openEnds > 0 {
			score += 10 // Potential threat
		} else if count == 1 && openEnds > 1 {
			score += 1 // Build opportunity
		}
	}
	
	return score
}

func (b *Bot) getLowestRow(game *Game, col int) int {
	for r := Rows - 1; r >= 0; r-- {
		if game.Board[r][col] == Empty {
			return r
		}
	}
	return -1
}

func (b *Bot) MakeMoveWithDelay(game *Game, gameServer *GameServer) {
	// Add slight delay to make it feel more natural
	delay := time.Duration(500+rand.Intn(1000)) * time.Millisecond
	time.Sleep(delay)
	
	col := b.GetBestMove(game)
	if col >= 0 {
		gameServer.handleMove(game, col, b.PlayerNum)
	}
}
