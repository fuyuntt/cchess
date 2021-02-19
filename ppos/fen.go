package ppos

import (
	"fmt"
	"github.com/fuyuntt/cchess/util"
	"regexp"
	"strings"
)

var pieceMap = map[int32]Piece{
	'k': PcBKing,
	'a': PcBAdvisor,
	'b': PcBBishop,
	'n': PcBKnight,
	'r': PcBRook,
	'c': PcBCannon,
	'p': PcBPawn,

	'K': PcRKing,
	'A': PcRAdvisor,
	'B': PcRBishop,
	'N': PcRKnight,
	'R': PcRRook,
	'C': PcRCannon,
	'P': PcRPawn,
}

const initFen = "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1"

var positionRegexp = regexp.MustCompile(`^(?:fen (?P<fen>[kabnrcpKABNRCP1-9/]+ [wrb] - - \d+ \d+)|(?P<startpos>startpos))(?: moves (?P<moves>[a-i]\d[a-i]\d(?: [a-i]\d[a-i]\d)*))?$`)

func parsePosition(positionStr string) (*Position, error) {
	groups := util.ParseGroup(positionRegexp, positionStr)
	var pos *Position
	for _, group := range groups {
		switch group.Key {
		case "fen":
			fenPos, err := parseFen(group.Value)
			if err != nil {
				return nil, err
			}
			pos = fenPos
		case "startpos":
			fenPos, err := parseFen(initFen)
			if err != nil {
				return nil, err
			}
			pos = fenPos
		case "moves":
			if pos == nil {
				continue
			}
			for _, mv := range strings.Split(group.Value, " ") {
				pos.MakeMove(GetMoveFromICCS(mv))
			}
		}
	}
	if pos == nil {
		return nil, fmt.Errorf("illegle positionStr: %s", positionStr)
	}
	return pos, nil
}
func parseFen(fenStr string) (*Position, error) {
	pos := CreatePosition()
	fenParts := strings.Split(fenStr, " ")
	x, y := 0, 0
	for _, b := range fenParts[0] {
		if b >= '0' && b <= '9' {
			x += int(b - '0')
		} else if b == '/' {
			y++
			x = 0
		} else {
			piece, ok := pieceMap[b]
			if !ok {
				return nil, fmt.Errorf("fen parse error: %s", fenStr)
			}
			pos.AddPiece(GetSquare(x, y), piece)
			x++
		}
	}
	side := fenParts[1]
	if side == "b" {
		pos.ChangeSide()
	}
	return pos, nil
}
