package cchess

import (
	"fmt"
	"sort"
	"time"
)

// 初始move列表长度
const initMoveCap = 128

// 最大深度
const limitDepth = 32

// 杀棋分
const mateValue = 10000

// 搜索出胜局的分数
const winValue = mateValue - 100

// 先行优势
const advancedValue = 3

func toBoolArr(origin [256]int) [256]bool {
	var res [256]bool
	for i := 0; i < 256; i++ {
		res[i] = origin[i] == 1
	}
	return res
}

type Piece int8

const (
	PcNop Piece = 0x00

	PcRKing    Piece = 0x08
	PcRAdvisor       = 0x09
	PcRBishop        = 0x0A
	PcRKnight        = 0x0B
	PcRRook          = 0x0C
	PcRCannon        = 0x0D
	PcRPawn          = 0x0E

	PcBKing    Piece = 0x10
	PcBAdvisor       = 0x11
	PcBBishop        = 0x12
	PcBKnight        = 0x13
	PcBRook          = 0x14
	PcBCannon        = 0x15
	PcBPawn          = 0x16
)

func GetPiece(pieceType PieceType, side Side) Piece {
	return Piece(side<<3) + Piece(pieceType)
}

func (pc Piece) GetSide() Side {
	return Side(pc >> 3)
}
func (pc Piece) GetType() PieceType {
	return PieceType(pc & 0x07)
}

type Move uint16

const MvNop Move = 0x0000

func (mv Move) Src() Square {
	return Square(mv & 0xff)
}
func (mv Move) Dst() Square {
	return Square(mv >> 8)
}
func (mv Move) String() string {
	return fmt.Sprintf("0x%x", int(mv))
}

func GetMove(src Square, dst Square) Move {
	return Move(dst<<8 + src)
}

type MoveSorter struct {
	moves []Move
	eval  func(mv Move) int
}

func (sorter MoveSorter) Len() int {
	return len(sorter.moves)
}
func (sorter MoveSorter) Less(i, j int) bool {
	// 逆序排序
	return sorter.eval(sorter.moves[j]) < sorter.eval(sorter.moves[i])
}
func (sorter MoveSorter) Swap(i, j int) {
	sorter.moves[i], sorter.moves[j] = sorter.moves[j], sorter.moves[i]
}

type PieceType int8

const (
	PtKing    PieceType = 0x00
	PtAdvisor           = 0x01
	PtBishop            = 0x02
	PtKnight            = 0x03
	PtRook              = 0x04
	PtCannon            = 0x05
	PtPawn              = 0x06
)

type Side int8

const (
	SdNop   Side = 0x00
	SdRed        = 0x01
	SdBlack      = 0x02
)

// 对方
func (site Side) OpSide() Side {
	return 0x03 - site
}

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
	value := pieceValue[pc.GetType()][sq]
	if side == SdRed {
		pos.vlRed += value
	} else if side == SdBlack {
		pos.vlBlack += value
	}
}
func (pos *Position) DelPiece(sq Square) Piece {
	pcCaptured := pos.pcSquares[sq]
	pos.pcSquares[sq] = PcNop
	side := pcCaptured.GetSide()
	value := pieceValue[pcCaptured.GetType()][sq]
	if side == SdRed {
		pos.vlRed -= value
	} else if side == SdBlack {
		pos.vlBlack -= value
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

// 创建局面
func CreatePosition() *Position {
	pos := &Position{}
	pos.playerSd = SdRed
	return pos
}

// 棋盘格子
type Square int

func (sq Square) GetX() int {
	return int(sq & 0x0f)
}
func (sq Square) GetY() int {
	return int(sq >> 4)
}
func (sq Square) String() string {
	return fmt.Sprintf("%2x", int(sq))
}
func GetSquare(x, y int) Square {
	return Square(y<<4 + x)
}

const (
	SqStart Square = 0x33
	SqEnd   Square = 0xcb
)

var sqInBoard = toBoolArr([256]int{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0,
	0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0,
	0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0,
	0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0,
	0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0,
	0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0,
	0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0,
	0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0,
	0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0,
	0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
})

func (sq Square) InBoard() bool {
	return sqInBoard[sq]
}

// 判断棋子是否在九宫的数组
var sqInFort = toBoolArr([256]int{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
})

func (sq Square) InFort() bool {
	return sqInFort[sq]
}
func (sq Square) GetSide() Side {
	return Side(2 - (sq >> 7))
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
