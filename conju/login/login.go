package login

import "math/rand"

// A LoginCode is a secret string we send to users as part of their
// Login link. It's stored as a string field in the Person object.
const loginCodeLength = 12

func RandomLoginCodeString() string {
	b := make([]rune, loginCodeLength)
	for i := range b {
		r := rand.Intn(36)
		if r < 10 {
			// 0..9
			b[i] = int32(r) + 48
		} else {
			// A..Z ((r - 10) + 65)
			b[i] = int32(r) + 55
		}
	}
	return string(b)
}
