package ucci

import (
	"fmt"
	"testing"
	"time"
)

func TestSearch(t *testing.T) {
	pos, err := parsePosition("startpos moves b2e2 h9g7 b0c2 b9a7 a0b0 b7f7 h0g2 a9b9 b0b9 a7b9 h2i2 b9c7 i0h0 i9i8 h0h6 f7e7 h6g6 i8g8 c3c4 h7h5 c2d4 c6c5 c4c5 h5e5 d4f5 e5e2 i2e2 g8f8 f5g7 f8f2 g2i1 e7e3 e2e6 f2e2 d0e1 e2c2 e0d0 c2c0 d0d1 c0c1 d1d0 c1c0 d0d1 c0c1 d1d0 c1c0 d0d1 c0c1 d1d0")
	if err != nil {
		t.Error(err)
	}
	move, vl := pos.SearchMain(3 * time.Second)
	fmt.Println(move, vl)
}
