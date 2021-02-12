package ucci

import (
	"fmt"
	"github.com/fuyuntt/cchess/ppos"
	"strings"
)

var pieceMap = map[int32]ppos.Piece{
	'k': ppos.PcBKing,
	'a': ppos.PcBAdvisor,
	'b': ppos.PcBBishop,
	'n': ppos.PcBKnight,
	'r': ppos.PcBRook,
	'c': ppos.PcBCannon,
	'p': ppos.PcBPawn,

	'K': ppos.PcRKing,
	'A': ppos.PcRAdvisor,
	'B': ppos.PcRBishop,
	'N': ppos.PcRKnight,
	'R': ppos.PcRRook,
	'C': ppos.PcRCannon,
	'P': ppos.PcRPawn,
}

const initFen = "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1"

//var positionRegexp = regexp.MustCompile(`^(?:(?P<fen>fen [kabnrcpKABNRCP1-9/]+ [wrb] - - \d+ \d+)|(?P<startpos>startpos))(?P<moves> moves( [a-i]\d[a-i]\d)+)?$`)
func parsePosition(positionStr string) (*ppos.Position, error) {
	parts := strings.Split(positionStr, " ")
	var pos *ppos.Position
	var i = 0
	for i < len(parts) {
		cmd := parts[i]
		switch cmd {
		case "fen":
			if len(parts) <= i+6 {
				return nil, fmt.Errorf("illegle fen: %s", positionStr)
			}
			var err error
			pos, err = parseFen(strings.Join(parts[1:7], " "))
			if err != nil {
				return nil, fmt.Errorf("fen parse failure: %s, err:%v", positionStr, err)
			}
			i += 7
		case "startpos":
			pos, _ = parseFen(initFen)
			i += 1
		case "moves":
			if pos == nil {
				return nil, fmt.Errorf("illegle positionStr: %s", positionStr)
			}
			for i++; i < len(parts); i++ {
				mv := parts[i]
				_, success := pos.MakeMove(ppos.GetMoveFromICCS(mv))
				if !success {
					return nil, fmt.Errorf("illegl move: %s", positionStr)
				}
			}
		}
	}
	return pos, nil
}

func parseFen(fenStr string) (*ppos.Position, error) {
	pos := ppos.CreatePosition()
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
			pos.AddPiece(ppos.GetSquare(x, y), piece)
			x++
		}
	}
	side := fenParts[1]
	if side == "b" {
		pos.ChangeSide()
	}
	return pos, nil
}
