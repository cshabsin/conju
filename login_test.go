package conju

import (
	"testing"
)

func TestRandomLoginCode(t *testing.T) {
	for k := 0; k < 100; k++ {
		lc := randomLoginCodeString()
		for i := 0; i < len(lc); i++ {
			r := lc[i]
			if r < 48 || (r > 57 && r < 65) || r > 90 {
				t.Errorf("Random Login code (%s) was incorrect at %d, got %d, out of range.",
					lc, i, r)
			}
		}
	}
}
