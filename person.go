package conju

// TODO: move to "package models"?

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/appengine/datastore"
)

type Person struct {
	FirstName     string
	LastName      string
	Nickname      string
	Email         string
	Telephone     string
	Address       string
	Birthdate     time.Time
	FallbackAge   float64
	NeedBirthdate bool
	// these fields can be removed after all the data is ported
	OldGuestId    int
	OldInviteeId  int
	OldInviteCode string
}

func PersonKey(ctx context.Context) *datastore.Key {
	return datastore.NewIncompleteKey(ctx, "Person", nil)
}

func CreatePerson(ctx context.Context, first, last string) error {
	p := Person{
		FirstName: first,
		LastName:  last,
	}

	_, err := datastore.Put(ctx, PersonKey(ctx), &p)
	return err
}

type NameFormality int

const (
	Informal NameFormality = iota // Chris Shabsin
	Formal                        // Christopher Shabsin
	Full                          // Christopher (Chris) Shabsin
)

func (p *Person) GetFirstName(formality NameFormality) string {
	if p.Nickname != "" && formality == Informal {
		return p.Nickname
	}
	return p.FirstName
}

func (p *Person) FullName() string {
	return p.FullNameWithFormality(Informal)
}

// FullName returns the formatted full name of the person, with
// nickname if present.
func (p *Person) FullNameWithFormality(formality NameFormality) string {

	if p.Nickname != "" && formality == Full {
		return fmt.Sprintf("%s (%s) %s", p.FirstName, p.Nickname, p.LastName)
	}

	return fmt.Sprintf("%s %s", p.GetFirstName(formality), p.LastName)

}

func CollectiveAddress(people []Person, formality NameFormality) string {
	//TODO: throw error here?
	if formality == Full {
		formality = Formal
	}
	if len(people) == 1 {
		return people[0].FullNameWithFormality(formality)
	}
	commonLastName := getCommonLastName(people)
	if len(people) == 2 {
		if commonLastName == "" {
			return fmt.Sprintf("%s & %s", people[0].FullNameWithFormality(formality), people[1].FullNameWithFormality(formality))
		} else {
			return fmt.Sprintf("%s & %s %s", people[0].GetFirstName(formality), people[1].GetFirstName(formality), commonLastName)
		}
	}

	toReturn := ""
	for i := 0; i < len(people); i++ {

		toReturn += people[i].GetFirstName(formality)
		if i < len(people)-2 {
			toReturn += ", "
		} else if i == len(people)-2 {
			toReturn += " & "
		}
	}
	if commonLastName != "" {
		toReturn += " " + commonLastName
	}
	return toReturn
}

func getCommonLastName(people []Person) string {
	var lastName string
	for i := 0; i < len(people); i++ {
		if lastName == "" {
			lastName = people[i].LastName
		} else if lastName != people[i].LastName {
			return ""
		}
	}
	return lastName
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
