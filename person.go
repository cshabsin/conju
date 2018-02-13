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
	Address     string
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

// Round a duration to half-years.
func HalfYears(d time.Duration) float64 {
	return RoundDuration(d, Halfyear).Hours() / 24 / 365
}
