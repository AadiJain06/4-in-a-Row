package game

import (
	"math/rand"
	"time"
)

// Bot is a simple but competitive opponent that tries to win,
// then block, then favor center columns.
type Bot struct {
	Player int
}

func NewBot(player int) *Bot {
	return &Bot{Player: player}
}

func (b *Bot) ChooseMove(board Board) int {
	rand.Seed(time.Now().UnixNano())

	// 1. Take winning move if available.
	if move, ok := findImmediate(board, b.Player); ok {
		return move
	}
	// 2. Block opponent winning move.
	opponent := CellP1
	if b.Player == CellP1 {
		opponent = CellP2
	}
	if move, ok := findImmediate(board, opponent); ok {
		return move
	}

	// 3. Prefer center columns to build threats.
	preferred := []int{3, 2, 4, 1, 5, 0, 6}
	for _, col := range preferred {
		if canPlay(board, col) {
			return col
		}
	}
	// 4. Fallback first available column.
	for col := 0; col < Columns; col++ {
		if canPlay(board, col) {
			return col
		}
	}
	return 0
}

func findImmediate(board Board, player int) (int, bool) {
	for col := 0; col < Columns; col++ {
		if !canPlay(board, col) {
			continue
		}
		tmp := CopyBoard(board)
		if res, _ := tmp.ApplyMove(col, player); res.Winner == player {
			return col, true
		}
	}
	return -1, false
}

func canPlay(board Board, col int) bool {
	return board[0][col] == CellEmpty
}

