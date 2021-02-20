// 9 +-+-+-+-k-+-+-+-+
// 8 +-+-+-+-+-+-+-+-+
// 7 +-+-+-+-P-+-+-+-+
// 6 +-+-+-+-+-+-+-+-+
// 5 +-+-+-+-+-+-+-+-+
// 4 +-+-+-+-+-+-+-+-+
// 3 +-+-+-+-+-+-+-+-+
// 2 +-+-+-+-+-+-+-+-+
// 1 +-+-+-+-+-+-+-+-+
// 0 +-+-+-+-+-K-+-+-+
//   a b c d e f g h i
package ppos

import (
	"fmt"
	"testing"
	"time"
)

func TestPosition(t *testing.T) {
	pos, _ := CreatePositionFromFenStr("4ka3/9/9/6N2/9/9/4P4/9/9/5K3 r - - 0 1")
	fmt.Println(pos.String())
	fmt.Println(pos.FenString())
	mv, vl := pos.SearchMain(3 * time.Second)
	if mv[0].String() != "e3e4" || vl < 9900 {
		t.Errorf("未搜索到杀棋, mv:%v, vl:%v", mv, vl)
	}
}
func TestPositionStart(t *testing.T) {
	pos, _ := CreatePositionFromPosStr("startpos")
	fmt.Println(pos.String())
	fmt.Println(pos.FenString())
	mv, vl := pos.SearchMain(3 * time.Second)
	fmt.Println(mv, vl)
}

func TestPositionTT(t *testing.T) {
	pos, _ := CreatePositionFromPosStr("fen 3PN4/4ak3/4Ra3/9/9/9/9/6n2/3p1p3/4KC1rc w - - 0 1")
	fmt.Println(pos.String())
	fmt.Println(pos.FenString())
	mv, vl := pos.SearchMain(3 * time.Second)
	fmt.Println(mv, vl)
}
func TestPositionBishop(t *testing.T) {
	pos := CreatePosition()
	pos.AddPiece(Square(0xa7), PcRBishop)
	pos.AddPiece(Square(0x85), PcBRook)
	pos.AddPiece(Square(0xc9), PcBPawn)
	mv, _ := pos.SearchMain(1 * time.Second)
	if mv[0] != 0x85a7 {
		t.Errorf("should capture rook, %v", mv)
	}
	pos.AddPiece(0x96, PcBBishop)
	mv, _ = pos.SearchMain(1 * time.Second)
	if mv[0] != 0xc9a7 {
		t.Errorf("should capture pawn, %v", mv)
	}
}

func TestMv(t *testing.T) {
	mv := GetMoveFromICCS("h5i6")
	fmt.Println(GetMove(0x47, 0x38))
	fmt.Println(mv.String())
}

func TestMatch(t *testing.T) {

}
