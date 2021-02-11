package ucci

import (
	"fmt"
	"github.com/fuyuntt/cchess/cchess"
	"strings"
)

var pieceMap = map[int32]cchess.Piece{
	'k': cchess.PcBKing,
	'a': cchess.PcBAdvisor,
	'b': cchess.PcBBishop,
	'n': cchess.PcBKnight,
	'r': cchess.PcBRook,
	'c': cchess.PcBCannon,
	'p': cchess.PcBPawn,

	'K': cchess.PcRKing,
	'A': cchess.PcRAdvisor,
	'B': cchess.PcRBishop,
	'N': cchess.PcRKnight,
	'R': cchess.PcRRook,
	'C': cchess.PcRCannon,
	'P': cchess.PcRPawn,
}

const initFen = "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1"

//var positionRegexp = regexp.MustCompile(`^(?:(?P<fen>fen [kabnrcpKABNRCP1-9/]+ [wrb] - - \d+ \d+)|(?P<startpos>startpos))(?P<moves> moves( [a-i]\d[a-i]\d)+)?$`)
func parsePosition(positionStr string) (*cchess.Position, error) {
	parts := strings.Split(positionStr, " ")
	var position *cchess.Position
	var i = 0
	for i < len(parts) {
		cmd := parts[i]
		switch cmd {
		case "fen":
			if len(parts) <= i+6 {
				return nil, fmt.Errorf("illegle fen: %s", positionStr)
			}
			pos, err := parseFen(strings.Join(parts[1:7], " "))
			if err != nil {
				return nil, fmt.Errorf("fen parse failure: %s, err:%v", positionStr, err)
			}
			position = pos
			i += 7
		case "startpos":
			position, _ = parseFen(initFen)
			i += 1
		case "moves":
			if position == nil {
				return nil, fmt.Errorf("illegle positionStr: %s", positionStr)
			}
			for i++; i < len(parts); i++ {
				mv := parts[i]
				srcX, srcY, dstX, dstY := int(3+mv[0]-'a'), int(3+mv[1]), int(3+mv[2]-'a'), int(3+mv[3])
				_, success := position.MakeMove(cchess.GetMove(cchess.GetSquare(srcX, srcY), cchess.GetSquare(dstX, dstY)))
				if !success {
					return nil, fmt.Errorf("illegl move: %s", positionStr)
				}
			}
		}
	}
	return position, nil
}

func parseFen(fenStr string) (*cchess.Position, error) {
	position := cchess.CreatePosition()
	fenParts := strings.Split(fenStr, " ")
	x, y := 3, 3
	for _, b := range fenParts[0] {
		if b >= '0' && b <= '9' {
			x += int(b - '0')
		} else if b == '/' {
			y++
			x = 3
		} else {
			piece, ok := pieceMap[b]
			if !ok {
				return nil, fmt.Errorf("fen parse error: %s", fenStr)
			}
			position.AddPiece(cchess.GetSquare(x, y), piece)
			x++
		}
	}
	side := fenParts[1]
	if side == "b" {
		position.ChangeSide()
	}
	return position, nil
}
