package ppos

import "testing"

func TestICCS(t *testing.T) {
	var suit = map[string]Move{
		"a0a1": GetMove(GetSquare(0, 9), GetSquare(0, 8)),
		"i0i9": GetMove(GetSquare(8, 9), GetSquare(8, 0)),
	}
	for k, v := range suit {
		iccs := GetMoveFromICCS(k)
		if v != iccs {
			t.Errorf("expact: %v, actual: %v", v, iccs)
		}
	}
}
