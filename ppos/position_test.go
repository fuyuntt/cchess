//00
//10
//20	                    27
//30		33	34	35	36+	37+	38+	39	3a	3b
//40		43	44	45	46+	47+	48+	49	4a	4b
//50		53	54	55	56+	57+	58+	59	5a	5b
//60		63	64	65	66	67	68	69	6a	6b
//70	72	73	74	75	76	77	78	79	7a	7b	7c
//80	82	83	84	85	86	87	88	89	8a	8b	8c
//90		93	94	95	96	97	98	99	9a	9b
//a0		a3	a4	a5	a6+	a7+	a8+	a9	aa	ab
//b0		b3	b4	b5	b6+	b7+	b8+	b9	ba	bb
//c0		c3	c4	c5	c6+	c7+	c8+	c9	ca	cb
//d0						d7
//e0
//f0
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
	mv, vl := pos.SearchMain(30 * time.Second)
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
