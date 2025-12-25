package game

import "errors"

const (
	Columns = 7
	Rows    = 6
)

const (
	CellEmpty = 0
	CellP1    = 1
	CellP2    = 2
)

var (
	ErrColumnFull   = errors.New("column is full")
	ErrInvalidTurn  = errors.New("not your turn")
	ErrInvalidCol   = errors.New("invalid column")
	ErrGameFinished = errors.New("game already finished")
)

type Board [Rows][Columns]int

type MoveResult struct {
	Board   Board
	Winner  int
	IsDraw  bool
	Winning [][2]int
}

func (b *Board) ApplyMove(col int, player int) (MoveResult, error) {
	if col < 0 || col >= Columns {
		return MoveResult{}, ErrInvalidCol
	}
	for row := Rows - 1; row >= 0; row-- {
		if b[row][col] == CellEmpty {
			b[row][col] = player
			return evaluate(*b, row, col, player), nil
		}
	}
	return MoveResult{}, ErrColumnFull
}

func evaluate(board Board, row, col, player int) MoveResult {
	directions := [][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}
	for _, d := range directions {
		coords := winningCoords(board, row, col, player, d[0], d[1])
		if len(coords) >= 4 {
			return MoveResult{Board: board, Winner: player, Winning: coords}
		}
	}

	isDraw := true
	for c := 0; c < Columns; c++ {
		if board[0][c] == CellEmpty {
			isDraw = false
			break
		}
	}
	return MoveResult{Board: board, IsDraw: isDraw}
}

func winningCoords(board Board, row, col, player, dx, dy int) [][2]int {
	coords := [][2]int{{row, col}}
	check := func(r, c int) {
		for r >= 0 && r < Rows && c >= 0 && c < Columns {
			if board[r][c] != player {
				return
			}
			coords = append(coords, [2]int{r, c})
			r += dx
			c += dy
		}
	}
	check(row+dx, col+dy)
	check(row-dx, col-dy)
	if len(coords) >= 4 {
		return coords
	}
	return [][2]int{}
}

func CopyBoard(src Board) Board {
	var dest Board
	for r := 0; r < Rows; r++ {
		for c := 0; c < Columns; c++ {
			dest[r][c] = src[r][c]
		}
	}
	return dest
}

