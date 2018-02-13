package conju

// TODO: move to "package models"?

import (
	"fmt"
	"time"
)

type Person struct {
	FirstName   string
	LastName    string
	Nickname    string
	Email       string
	HomePhone   string
	MobilePhone string
	Birthdate   time.Time
	FallbackAge float64
}

// FullName returns the formatted full name of the person, with
// nickname if present.
func (p *Person) FullName() string {
	if p.Nickname != "" {
		return fmt.Sprintf("%s (%s) %s", p.FirstName, p.Nickname, p.LastName)
	} else {
		return fmt.Sprintf("%s %s", p.FirstName, p.LastName)
	}
}

const (
	Halfyear time.Duration = 12 * 365 * time.Hour
	Year                   = 2 * Halfyear
)

// Returns the age of the person
func (p Person) ApproxAge() time.Duration {
	if p.Birthdate.IsZero() {
		return time.Duration(p.FallbackAge) * Year
	}
	return time.Now().Sub(p.Birthdate)
}

// Cribbed from Go 1.9 library -------------------------------
const (
	minDuration time.Duration = -1 << 63
	maxDuration time.Duration = 1<<63 - 1
)

// lessThanHalf reports whether x+x < y but avoids overflow,
// assuming x and y are both positive (Duration is signed).
func lessThanHalf(x, y time.Duration) bool {
	return uint64(x)+uint64(x) < uint64(y)
}

// Round returns the result of rounding d to the nearest multiple of m.
// The rounding behavior for halfway values is to round away from zero.
// If the result exceeds the maximum (or minimum)
// value that can be stored in a Duration,
// Round returns the maximum (or minimum) duration.
// If m <= 0, Round returns d unchanged.
func RoundDuration(d time.Duration, m time.Duration) time.Duration {
	if m <= 0 {
		return d
	}
	r := d % m
	if d < 0 {
		r = -r
		if lessThanHalf(r, m) {
			return d + r
		}
		if d1 := d - m + r; d1 < d {
			return d1
		}
		return minDuration // overflow
	}
	if lessThanHalf(r, m) {
		return d - r
	}
	if d1 := d + m - r; d1 > d {
		return d1
	}
	return maxDuration // overflow
}

// Round a duration to half-years.
func HalfYears(d time.Duration) float64 {
	return RoundDuration(d, Halfyear).Hours() / 24 / 365
}
