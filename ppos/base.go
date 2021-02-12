package ppos

import "fmt"

type Piece int8

const (
	PcNop Piece = 0x00

	PcRKing    Piece = 0x08
	PcRAdvisor Piece = 0x09
	PcRBishop  Piece = 0x0A
	PcRKnight  Piece = 0x0B
	PcRRook    Piece = 0x0C
	PcRCannon  Piece = 0x0D
	PcRPawn    Piece = 0x0E

	PcBKing    Piece = 0x10
	PcBAdvisor Piece = 0x11
	PcBBishop  Piece = 0x12
	PcBKnight  Piece = 0x13
	PcBRook    Piece = 0x14
	PcBCannon  Piece = 0x15
	PcBPawn    Piece = 0x16
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
func (pc Piece) String() string {
	pcType := pc.GetType()
	if pc.GetSide() == SdRed {
		return "kabnrcp"[pcType : pcType+1]
	} else if pc.GetSide() == SdBlack {
		return "KABNRCP"[pcType : pcType+1]
	} else {
		return "+"
	}
}

type PieceType int8

const (
	PtKing    PieceType = 0x00
	PtAdvisor PieceType = 0x01
	PtBishop  PieceType = 0x02
	PtKnight  PieceType = 0x03
	PtRook    PieceType = 0x04
	PtCannon  PieceType = 0x05
	PtPawn    PieceType = 0x06
)

type Side int8

const (
	SdRed   Side = 0x01
	SdBlack Side = 0x02
)

// 对方
func (site Side) OpSide() Side {
	return 0x03 - site
}
func (site Side) String() string {
	switch site {
	case SdRed:
		return "Red"
	case SdBlack:
		return "Black"
	default:
		return "Nop"
	}
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
	return mv.ICCS()
}
func (mv Move) ICCS() string {
	return string([]rune{
		rune('a' + mv.Src().GetX()),
		rune('0' + 9 - mv.Src().GetY()),
		rune('a' + mv.Dst().GetX()),
		rune('0' + 9 - mv.Dst().GetY())},
	)
}

func GetMove(src Square, dst Square) Move {
	return Move(dst<<8 + src)
}
func GetMoveFromICCS(iccs string) Move {
	srcX, srcY, dstX, dstY := iccsToX(iccs[0]), iccsToY(iccs[1]), iccsToX(iccs[2]), iccsToY(iccs[3])
	return GetMove(GetSquare(srcX, srcY), GetSquare(dstX, dstY))
}
func iccsToX(c byte) int {
	return int(c - 'a')
}
func iccsToY(c byte) int {
	return int(9 - (c - '0'))
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

// 棋盘格子
type Square int

func (sq Square) GetX() int {
	return int(sq&0x0f) - 3
}
func (sq Square) GetY() int {
	return int(sq>>4) - 3
}
func (sq Square) String() string {
	return fmt.Sprintf("%2x", int(sq))
}

// 翻转在棋盘的位置
func (sq Square) Flip() Square {
	return 0xfe - sq
}
func GetSquare(x, y int) Square {
	x += 3
	y += 3
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

func toBoolArr(origin [256]int) [256]bool {
	var res [256]bool
	for i := 0; i < 256; i++ {
		res[i] = origin[i] == 1
	}
	return res
}
