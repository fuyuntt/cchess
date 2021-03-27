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
	pos, _ := CreatePositionFromPosStr("startpos moves b2e2 b9c7 b0c2 a9b9 a0b0 h9g7 b0b4 i9i8 h2f2 i8f8 f0e1 g6g5 g3g4 g5g4 b4g4 h7h3 c3c4 b7a7 h0g2 h3h5 g2f4 h5f5 f4d5 f8c8 i0h0 b9b5 d5f6 c6c5 i3i4 c8f8 a3a4 c5c4 g4c4 c7d5 c4g4 a7c7 c0a2 f8f7 f6h7 f5e5 h7g9 f7e7 h0h7 g7f5 h7e7 c9e7 g9i8 e5e2 g0e2 f9e8 a2c0 b5c5 c2b4 d5e3 g4f4 c7c8 b4a6 c5e5 f2h2 c8i8 h2h9 i8f8 f4d4 e3g2 d4g4 g2f4 g4g9 f8f9 g9g4")
	fmt.Println(pos.String())
	//pos, _ := CreatePositionFromFenStr("3akc1C1/4a4/4b4/N3p3p/4rn3/P4nR1P/9/4B4/4A4/2BAK4 b - - 0 1")
	main, i := pos.SearchMain(3 * time.Second)
	fmt.Println(main, i)
	fmt.Println(pos.FenString())

}
