package cchess

import (
	"sort"
	"strings"
	"time"
)

// 最大深度
const limitDepth = 32

// 杀棋分
const mateValue = 10000

// 搜索出胜局的分数
const winValue = mateValue - 100

// 先行优势
const advancedValue = 3

type SearchCtx struct {
	// 电脑走的棋
	mvResult Move

	moves []Move

	// 距离根节点的步数
	nDistance int

	// 历史表
	historyTable [65536]int
}

type Position struct {
	// 棋盘
	pcSquares [256]Piece
	// 该哪方走
	playerSd Side
	// 红棋分
	vlRed int
	// 黑旗分
	vlBlack int
}

func (pos *Position) ChangeSide() {
	pos.playerSd = pos.playerSd.OpSide()
}
func (pos *Position) AddPiece(sq Square, pc Piece) {
	pos.pcSquares[sq] = pc
	side := pc.GetSide()
	pcValue := pieceValue[pc.GetType()]
	if side == SdRed {
		pos.vlRed += pcValue[sq]
	} else if side == SdBlack {
		pos.vlBlack += pcValue[sq.Flip()]
	}
}
func (pos *Position) DelPiece(sq Square) Piece {
	pcCaptured := pos.pcSquares[sq]
	pos.pcSquares[sq] = PcNop
	side := pcCaptured.GetSide()
	pcValueTable := pieceValue[pcCaptured.GetType()]
	if side == SdRed {
		pos.vlRed -= pcValueTable[sq]
	} else if side == SdBlack {
		pos.vlBlack -= pcValueTable[sq.Flip()]
	}
	return pcCaptured
}

func (pos *Position) Evaluate() int {
	if pos.playerSd == SdRed {
		return pos.vlRed - pos.vlBlack + advancedValue
	} else {
		return pos.vlBlack - pos.vlRed + advancedValue
	}
}
func (pos *Position) MovePiece(mv Move) Piece {
	var sqSrc, sqDst = mv.Src(), mv.Dst()
	var pcSrc, pcDst = pos.DelPiece(sqSrc), pos.DelPiece(sqDst)
	pos.AddPiece(sqDst, pcSrc)
	return pcDst
}
func (pos *Position) UndoMovePiece(mv Move, pcCaptured Piece) {
	var sqSrc, sqDst = mv.Src(), mv.Dst()
	var _, pcDst = pos.DelPiece(sqSrc), pos.DelPiece(sqDst)
	pos.AddPiece(sqSrc, pcDst)
	pos.AddPiece(sqDst, pcCaptured)
}
func (pos *Position) Checked() bool {
	sqSrc := SqStart
	selfKing := GetPiece(PtKing, pos.playerSd)
	for sqSrc = SqStart; sqSrc <= SqEnd && pos.pcSquares[sqSrc] != selfKing; sqSrc++ {
	}
	if sqSrc > SqEnd {
		return false
	}
	opSide := pos.playerSd.OpSide()
	// 1. 判断是否被对方的兵(卒)将军
	if pos.pcSquares[sqForward(sqSrc, pos.playerSd)] == GetPiece(PtPawn, opSide) {
		return true
	}
	for delta := Square(-0x01); delta <= 0x01; delta += 0x02 {
		if pos.pcSquares[sqSrc+delta] == GetPiece(PtPawn, opSide) {
			return true
		}
	}

	// 2. 判断是否被对方的马将军
	opKnight := GetPiece(PtKnight, opSide)
	for i := 0; i < 8; i++ {
		sqDst := sqSrc + knightMoveTab[i]
		if pos.pcSquares[sqDst] == opKnight && pos.pcSquares[getKnightPin(sqDst, sqSrc)] == PcNop {
			return true
		}
	}
	// 3. 判断是否被对方的车或炮将军(包括将帅对脸)
	opRook := GetPiece(PtRook, opSide)
	opCannon := GetPiece(PtCannon, opSide)
	opKing := GetPiece(PtKing, opSide)
	for i := 0; i < 4; i++ {
		delta := lineMoveDelta[i]
		sqDst := sqSrc + delta
		for ; sqDst.InBoard() && pos.pcSquares[sqDst] == PcNop; sqDst += delta {
		}
		if !sqDst.InBoard() {
			continue
		}
		pcDst := pos.pcSquares[sqDst]
		if pcDst == opRook || pcDst == opKing {
			return true
		}
		sqDst += delta
		for ; sqDst.InBoard() && pos.pcSquares[sqDst] == PcNop; sqDst += delta {
		}
		if !sqDst.InBoard() {
			continue
		}
		if pos.pcSquares[sqDst] == opCannon {
			return true
		}
	}
	return false
}
func (pos *Position) GenerateMoves(moves []Move) []Move {
	// 生成所有走法
	//	int i, j, nGenMoves, nDelta, sqSrc, sqDst;
	//	int pcSelfSide, pcOppSide, pcSrc, pcDst;
	// 生成所有走法，需要经过以下几个步骤：

	for sqSrc := SqStart; sqSrc <= SqEnd; sqSrc++ {
		pcSrc := pos.pcSquares[sqSrc]
		if pcSrc.GetSide() != pos.playerSd {
			continue
		}
		switch pcSrc.GetType() {
		case PtKing:
			for i := 0; i < 4; i++ {
				var sqDst = sqSrc + kingMoveTab[i]
				if !sqDst.InFort() {
					continue
				}
				pcDst := pos.pcSquares[sqDst]
				if pcDst.GetSide() != pos.playerSd {
					moves = append(moves, GetMove(sqSrc, sqDst))
				}
			}
		case PtAdvisor:
			for i := 0; i < 4; i++ {
				sqDst := sqSrc + advisorMoveTab[i]
				if !sqDst.InFort() {
					continue
				}
				pcDst := pos.pcSquares[sqDst]
				if pcDst.GetSide() != pos.playerSd {
					moves = append(moves, GetMove(sqSrc, sqDst))
				}
			}
		case PtBishop:
			for i := 0; i < 4; i++ {
				sqDst := sqSrc + bishopMoveTab[i]
				sqBishopPin := (sqSrc + sqDst) >> 1
				if sqDst.InBoard() && sqDst.GetSide() == pos.playerSd &&
					pos.pcSquares[sqBishopPin] == PcNop && pos.pcSquares[sqDst].GetSide() != pos.playerSd {
					moves = append(moves, GetMove(sqSrc, sqDst))
				}
			}
		case PtKnight:
			for i := 0; i < 8; i++ {
				sqDst := sqSrc + knightMoveTab[i]
				if !sqDst.InBoard() {
					continue
				}
				sqPin := getKnightPin(sqSrc, sqDst)
				if pos.pcSquares[sqPin] != PcNop {
					continue
				}
				if pos.pcSquares[sqDst].GetSide() == pos.playerSd {
					continue
				}
				moves = append(moves, GetMove(sqSrc, sqDst))
			}
		case PtRook:
			for i := 0; i < 4; i++ {
				sqDelta := lineMoveDelta[i]
				for sqDst := sqSrc + sqDelta; sqDst.InBoard(); sqDst += sqDelta {
					pcDst := pos.pcSquares[sqDst]
					if pcDst == PcNop {
						moves = append(moves, GetMove(sqSrc, sqDst))
					} else {
						if pcDst.GetSide() != pos.playerSd {
							moves = append(moves, GetMove(sqSrc, sqDst))
						}
						break
					}
				}
			}
		case PtCannon:
			for i := 0; i < 4; i++ {
				sqDelta := lineMoveDelta[i]
				sqDst := sqSrc + sqDelta
				for ; sqDst.InBoard(); sqDst += sqDelta {
					pcDst := pos.pcSquares[sqDst]
					if pcDst == PcNop {
						moves = append(moves, GetMove(sqSrc, sqDst))
					} else {
						break
					}
				}
				sqDst += sqDelta
				for ; sqDst.InBoard(); sqDst += sqDelta {
					pcDst := pos.pcSquares[sqDst]
					if pcDst != PcNop {
						if pcDst.GetSide() != pos.playerSd {
							moves = append(moves, GetMove(sqSrc, sqDst))
						}
						break
					}
				}
			}
		case PtPawn:
			sqDst := sqForward(sqSrc, pos.playerSd)
			if sqDst.InBoard() && pos.pcSquares[sqDst].GetSide() != pos.playerSd {
				moves = append(moves, GetMove(sqSrc, sqDst))
			}
			if sqSrc.GetSide() != pos.playerSd {
				for delta := Square(-0x01); delta <= 0x01; delta += 0x02 {
					sqDst = sqSrc + delta
					if sqDst.InBoard() && pos.pcSquares[sqDst].GetSide() != pos.playerSd {
						moves = append(moves, GetMove(sqSrc, sqDst))
					}
				}
			}
		}
	}
	return moves
}

// 走棋 会变更当前走棋方
func (pos *Position) MakeMove(move Move) (Piece, bool) {
	pcCaptured := pos.MovePiece(move)
	if pos.Checked() {
		pos.UndoMovePiece(move, pcCaptured)
		return PcNop, false
	}
	pos.ChangeSide()
	return pcCaptured, true
}
func (pos *Position) UndoMakeMove(move Move, pcCaptured Piece) {
	pos.ChangeSide()
	pos.UndoMovePiece(move, pcCaptured)
}

func (pos *Position) searchAlphaBeta(searchCtx *SearchCtx, vlAlpha int, vlBeta int, depth int) int {
	if depth == 0 {
		return pos.Evaluate()
	}
	moves := searchCtx.moves[:0]
	vlBest := -mateValue
	var mvBest = MvNop
	moves = pos.GenerateMoves(moves)
	searchCtx.moves = moves
	sort.Sort(MoveSorter{moves: moves, eval: func(mv Move) int {
		return searchCtx.historyTable[mv]
	}})
	for _, mv := range moves {
		pcCaptured, success := pos.MakeMove(mv)
		if !success {
			continue
		}
		searchCtx.nDistance++
		vl := -pos.searchAlphaBeta(searchCtx, -vlBeta, -vlAlpha, depth-1)
		pos.UndoMakeMove(mv, pcCaptured)
		searchCtx.nDistance--
		if vl > vlBest {
			vlBest = vl
			if vl >= vlBeta {
				mvBest = mv
				break
			}
			if vl > vlAlpha {
				mvBest = mv
				vlAlpha = vl
			}
		}
	}
	// 所有的move都无法走 杀棋!
	if vlBest == -mateValue {
		return searchCtx.nDistance - mateValue
	}
	if mvBest != MvNop {
		searchCtx.historyTable[mvBest] += depth * depth
		if searchCtx.nDistance == 0 {
			searchCtx.mvResult = mvBest
		}
	}
	return vlBest
}

func (pos *Position) SearchMain(duration time.Duration) (Move, int) {
	t := time.Now()
	searchCtx := &SearchCtx{}
	vl := 0
	for i := 0; i < limitDepth; i++ {
		vl = pos.searchAlphaBeta(searchCtx, -mateValue, mateValue, i)
		if vl > winValue || vl < -winValue {
			break
		}
		if time.Now().Sub(t) > duration {
			break
		}
	}
	return searchCtx.mvResult, vl
}
func (pos *Position) String() string {
	var sb strings.Builder
	for i := 0; i < 10; i++ {
		if i == 4 || i == 5 {
			sb.WriteRune('-')
		} else {
			sb.WriteRune(' ')
		}
		sb.WriteString(pos.pcSquares[GetSquare(3, i+3)].String())
		for j := 1; j < 9; j++ {
			sb.WriteRune('-')
			sb.WriteString(pos.pcSquares[GetSquare(j+3, i+3)].String())
		}
		if i == 4 || i == 5 {
			sb.WriteRune('-')
		} else {
			sb.WriteRune(' ')
		}
		sb.WriteRune('\n')
	}
	return sb.String()
}

// 创建局面
func CreatePosition() *Position {
	pos := &Position{}
	pos.playerSd = SdRed
	return pos
}

var kingMoveTab = [4]Square{-0x10, -0x01, +0x01, +0x10}
var lineMoveDelta = [4]Square{-0x10, -0x01, +0x01, +0x10}
var advisorMoveTab = [4]Square{-0x11, -0x0f, +0x0f, +0x11}
var bishopMoveTab = [4]Square{-0x22, -0x1e, +0x1e, +0x22}
var knightMoveTab = [8]Square{-0x21, -0x1f, -0x12, -0x0e, +0x0e, +0x12, +0x1f, +0x21}
var knightMovePinTab = [512]Square{
	0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, -0x10, 0, -0x10, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, -0x01, 0, 0, 0, +0x01, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, -0x01, 0, 0, 0, +0x01, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0x10, 0, 0x10, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0,
}

func getKnightPin(sqSrc Square, sqDst Square) Square {
	return sqSrc + knightMovePinTab[256+sqDst-sqSrc]
}
func sqForward(sq Square, sd Side) Square {
	return sq + Square((sd>>1)<<5-0x10)
}

// 子力位置价值表
var pieceValue = [7][256]int{
	{ // 帅(将)
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 11, 15, 11, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}, { // 仕(士)
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 20, 0, 20, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 23, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 20, 0, 20, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}, { // 相(象)
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 20, 0, 0, 0, 20, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 18, 0, 0, 0, 23, 0, 0, 0, 18, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 20, 0, 0, 0, 20, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}, { // 马
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 90, 90, 90, 96, 90, 96, 90, 90, 90, 0, 0, 0, 0,
		0, 0, 0, 90, 96, 103, 97, 94, 97, 103, 96, 90, 0, 0, 0, 0,
		0, 0, 0, 92, 98, 99, 103, 99, 103, 99, 98, 92, 0, 0, 0, 0,
		0, 0, 0, 93, 108, 100, 107, 100, 107, 100, 108, 93, 0, 0, 0, 0,
		0, 0, 0, 90, 100, 99, 103, 104, 103, 99, 100, 90, 0, 0, 0, 0,
		0, 0, 0, 90, 98, 101, 102, 103, 102, 101, 98, 90, 0, 0, 0, 0,
		0, 0, 0, 92, 94, 98, 95, 98, 95, 98, 94, 92, 0, 0, 0, 0,
		0, 0, 0, 93, 92, 94, 95, 92, 95, 94, 92, 93, 0, 0, 0, 0,
		0, 0, 0, 85, 90, 92, 93, 78, 93, 92, 90, 85, 0, 0, 0, 0,
		0, 0, 0, 88, 85, 90, 88, 90, 88, 90, 85, 88, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}, { // 车
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 206, 208, 207, 213, 214, 213, 207, 208, 206, 0, 0, 0, 0,
		0, 0, 0, 206, 212, 209, 216, 233, 216, 209, 212, 206, 0, 0, 0, 0,
		0, 0, 0, 206, 208, 207, 214, 216, 214, 207, 208, 206, 0, 0, 0, 0,
		0, 0, 0, 206, 213, 213, 216, 216, 216, 213, 213, 206, 0, 0, 0, 0,
		0, 0, 0, 208, 211, 211, 214, 215, 214, 211, 211, 208, 0, 0, 0, 0,
		0, 0, 0, 208, 212, 212, 214, 215, 214, 212, 212, 208, 0, 0, 0, 0,
		0, 0, 0, 204, 209, 204, 212, 214, 212, 204, 209, 204, 0, 0, 0, 0,
		0, 0, 0, 198, 208, 204, 212, 212, 212, 204, 208, 198, 0, 0, 0, 0,
		0, 0, 0, 200, 208, 206, 212, 200, 212, 206, 208, 200, 0, 0, 0, 0,
		0, 0, 0, 194, 206, 204, 212, 200, 212, 204, 206, 194, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}, { // 炮
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 100, 100, 96, 91, 90, 91, 96, 100, 100, 0, 0, 0, 0,
		0, 0, 0, 98, 98, 96, 92, 89, 92, 96, 98, 98, 0, 0, 0, 0,
		0, 0, 0, 97, 97, 96, 91, 92, 91, 96, 97, 97, 0, 0, 0, 0,
		0, 0, 0, 96, 99, 99, 98, 100, 98, 99, 99, 96, 0, 0, 0, 0,
		0, 0, 0, 96, 96, 96, 96, 100, 96, 96, 96, 96, 0, 0, 0, 0,
		0, 0, 0, 95, 96, 99, 96, 100, 96, 99, 96, 95, 0, 0, 0, 0,
		0, 0, 0, 96, 96, 96, 96, 96, 96, 96, 96, 96, 0, 0, 0, 0,
		0, 0, 0, 97, 96, 100, 99, 101, 99, 100, 96, 97, 0, 0, 0, 0,
		0, 0, 0, 96, 97, 98, 98, 98, 98, 98, 97, 96, 0, 0, 0, 0,
		0, 0, 0, 96, 96, 97, 99, 99, 99, 97, 96, 96, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}, { // 兵(卒)
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 9, 9, 9, 11, 13, 11, 9, 9, 9, 0, 0, 0, 0,
		0, 0, 0, 19, 24, 34, 42, 44, 42, 34, 24, 19, 0, 0, 0, 0,
		0, 0, 0, 19, 24, 32, 37, 37, 37, 32, 24, 19, 0, 0, 0, 0,
		0, 0, 0, 19, 23, 27, 29, 30, 29, 27, 23, 19, 0, 0, 0, 0,
		0, 0, 0, 14, 18, 20, 27, 29, 27, 20, 18, 14, 0, 0, 0, 0,
		0, 0, 0, 7, 0, 13, 0, 16, 0, 13, 0, 7, 0, 0, 0, 0,
		0, 0, 0, 7, 0, 7, 0, 15, 0, 7, 0, 7, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	},
}
