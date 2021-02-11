package cchess

import (
	"fmt"
	"testing"
	"time"
)

func TestPosition(t *testing.T) {
	pos := CreatePosition()
	pos.AddPiece(Square(0x37), PcBKing)
	pos.AddPiece(Square(0x38), PcBAdvisor)
	pos.AddPiece(Square(0xc8), PcRKing)
	pos.AddPiece(Square(0x97), PcRPawn)
	pos.AddPiece(Square(0x69), PcRKnight)
	fmt.Println(pos.String())
	mv, vl := pos.SearchMain(30 * time.Second)
	fmt.Println(mv.String(), vl)
}
