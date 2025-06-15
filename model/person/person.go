package person

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cshabsin/conju/conju/util"

	"cloud.google.com/go/datastore"
)

type Person struct {
	DatastoreKey     *datastore.Key
	FirstName        string
	LastName         string
	Nickname         string
	Pronouns         PronounSet
	Email            string
	Telephone        string
	Address          string
	Birthdate        time.Time
	FoodRestrictions []FoodRestriction
	FoodNotes        string
	IsAdmin          bool
	FallbackAge      float64
	NeedBirthdate    bool
	PrivateComments  string
	LoginCode        string
	// these fields can be removed after all the data is ported
	OldGuestId    int
	OldInviteeId  int
	OldInviteCode string
}

type PersonWithKey struct {
	Key    string
	Person Person
}

type PersonUpdateFormInfo struct {
	ThisPerson               *Person
	EncodedKey               string
	AllPronouns              []PronounSet
	AllFoodRestrictions      []FoodRestrictionTag
	HighlightNeededBirthdate bool
	PersonIndex              int
}

func MakePersonUpdateFormInfo(key *datastore.Key, person Person, index int, highlightNeededBirthdate bool) PersonUpdateFormInfo {
	encodedKey := ""
	if key != nil {
		*person.DatastoreKey = *key
		encodedKey = key.Encode()
	}

	return PersonUpdateFormInfo{
		ThisPerson:               &person,
		EncodedKey:               encodedKey,
		AllPronouns:              []PronounSet{They, She, He, Zie},
		AllFoodRestrictions:      GetAllFoodRestrictionTags(),
		HighlightNeededBirthdate: highlightNeededBirthdate,
		PersonIndex:              index,
	}
}

// TODO: define map of int -> string for pronoun enum --> display string

func PersonKey(ctx context.Context) *datastore.Key {
	return datastore.IncompleteKey("Person", nil)
}

// func CreatePerson(ctx context.Context, first, last string) error {
// 	p := Person{
// 		FirstName: first,
// 		LastName:  last,
// 		LoginCode: login.RandomLoginCodeString(),
// 	}

// 	_, err := datastore.Put(ctx, PersonKey(ctx), &p)
// 	return err
// }

type NameFormality int

const (
	Informal NameFormality = iota // Chris Shabsin
	Formal                        // Christopher Shabsin
	Full                          // Christopher (Chris) Shabsin
)

// If you change this also change GetPronouns
type PronounSet int

const (
	They PronounSet = iota
	She
	He
	Zie
)

func PronounFromConstant(pronounConstant int) PronounSet {
	return PronounSet(pronounConstant)
}

func GetPronouns(p PronounSet) string {
	switch p {
	case She:
		return "She/Her/Hers"
	case He:
		return "He/Him/His"
	case Zie:
		return "Zie/Hir/Hirs"
	default:
		return "They/Them/Theirs"
	}
}

type FoodRestriction int

const (
	Vegetarian FoodRestriction = iota
	Vegan
	NoRedMeat
	VegetarianPlusFish
	NoDairy
	NoGluten
	Kosher
	Halal
	InconvenientAllergy
	DangerousAllergy
)

type FoodRestrictionTag struct {
	Tag          FoodRestriction
	Description  string
	Supplemental string
}

func GetAllFoodRestrictionTags() []FoodRestrictionTag {
	var toReturn []FoodRestrictionTag

	toReturn = append(toReturn, FoodRestrictionTag{
		Tag:         Vegetarian,
		Description: "Vegetarian",
	})
	toReturn = append(toReturn, FoodRestrictionTag{
		Tag:         Vegan,
		Description: "Vegan",
	})
	toReturn = append(toReturn, FoodRestrictionTag{
		Tag:          NoRedMeat,
		Description:  "No Red Meat",
		Supplemental: "Chicken/Fish Okay",
	})
	toReturn = append(toReturn, FoodRestrictionTag{
		Tag:         VegetarianPlusFish,
		Description: "Vegetarian + Fish",
	})
	toReturn = append(toReturn, FoodRestrictionTag{
		Tag:         NoDairy,
		Description: "No Dairy",
	})
	toReturn = append(toReturn, FoodRestrictionTag{
		Tag:         NoGluten,
		Description: "No Gluten",
	})
	toReturn = append(toReturn, FoodRestrictionTag{
		Tag:         Kosher,
		Description: "Kosher",
	})
	toReturn = append(toReturn, FoodRestrictionTag{
		Tag:         Halal,
		Description: "Halal",
	})
	toReturn = append(toReturn, FoodRestrictionTag{
		Tag:          InconvenientAllergy,
		Description:  "Inconvenient Allergy",
		Supplemental: "List your allergies below.",
	})
	toReturn = append(toReturn, FoodRestrictionTag{
		Tag:          DangerousAllergy,
		Description:  "Dangerous Allergy",
		Supplemental: "List your allergies below.",
	})

	return toReturn
}

func (p Person) GetFirstName(formality NameFormality) string {
	if p.Nickname != "" && formality == Informal {
		return p.Nickname
	}
	return p.FirstName
}

func (p Person) FullName() string {
	return p.FullNameWithFormality(Informal)
}

func (p Person) FullNameWithAge(t time.Time) string {
	ageString := p.AgeString(t)
	if len(ageString) > 0 {
		return p.FullName() + " (" + ageString + ")"
	}
	return p.FullName()
}

func (p Person) FirstNameWithAge(t time.Time) string {
	ageString := p.AgeString(t)
	if len(ageString) > 0 {
		return p.GetFirstName(Informal) + " (" + ageString + ")"
	}
	return p.GetFirstName(Informal)
}

func (p Person) AgeString(t time.Time) string {
	if p.Birthdate.IsZero() {
		if !p.NeedBirthdate {
			return ""
		}
		if p.FallbackAge == 0 {
			return "???"
		}
		return fmt.Sprintf("%.1f", p.FallbackAge)
	}
	age := HalfYears(p.ApproxAgeAtTime(t))

	if age >= 16 {
		return ""
	}
	return fmt.Sprintf("%.1f", age)
}

// FullName returns the formatted full name of the person, with
// nickname if present.
func (p Person) FullNameWithFormality(formality NameFormality) string {

	if p.Nickname != "" && formality == Full {
		return fmt.Sprintf("%s (%s) %s", p.FirstName, p.Nickname, p.LastName)
	}

	return fmt.Sprintf("%s %s", p.GetFirstName(formality), p.LastName)

}

func CollectiveAddressFirstNames(people []Person, formality NameFormality) string {
	//TODO: throw error here?
	if formality == Full {
		formality = Formal
	}
	if len(people) == 1 {
		return people[0].GetFirstName(formality)
	}
	if len(people) == 2 {
		return people[0].GetFirstName(formality) + " & " + people[1].GetFirstName(formality)
	}

	toReturn := ""
	for i := 0; i < len(people); i++ {
		toReturn += people[i].GetFirstName(formality)
		if i < len(people)-2 {
			toReturn += ", "
		}
		if i == len(people)-2 {
			toReturn += " & "
		}
	}

	return toReturn
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

func SortByFirstName(a, b Person) bool {
	return strings.Compare(a.GetFirstName(Informal), b.GetFirstName(Informal)) < 0
}

func SortByLastFirstName(a, b Person) bool {
	lastNameComparison := strings.Compare(a.LastName, b.LastName)
	if lastNameComparison != 0 {
		return lastNameComparison < 0
	}
	return strings.Compare(a.FirstName, b.FirstName) < 0
}

//const AgeOfAdulthood 16

const (
	Halfyear time.Duration = 12 * 365 * time.Hour
	Year                   = 2 * Halfyear
)

// Returns the age of the person
func (p Person) ApproxAge() time.Duration {
	return p.ApproxAgeAtTime(time.Now())
}

func (p Person) ApproxAgeAtTime(dateTime time.Time) time.Duration {
	if p.Birthdate.IsZero() {
		if p.NeedBirthdate {
			return time.Duration(p.FallbackAge * 1000 * 1000 * 1000 * 60 * 24 * 365)
		} else {
			return 0
		}
	}
	return dateTime.Sub(p.Birthdate)
}

func (p Person) IsNonAdultAtTime(datetime time.Time) bool {
	if p.Birthdate.IsZero() {
		return p.NeedBirthdate
	}
	age := HalfYears(p.ApproxAgeAtTime(datetime))
	return age < 16
}

func (p Person) IsAdultAtTime(datetime time.Time) bool {
	if p.Birthdate.IsZero() {
		return !p.NeedBirthdate
	}
	age := HalfYears(p.ApproxAgeAtTime(datetime))
	return age >= 16
}

func (p Person) IsChildAtTime(datetime time.Time) bool {
	if p.Birthdate.IsZero() {
		return p.NeedBirthdate
	}
	age := HalfYears(p.ApproxAgeAtTime(datetime))
	return age < 16 && age >= 4
}

func (p Person) IsBabyAtTime(datetime time.Time) bool {
	if p.Birthdate.IsZero() {
		if p.NeedBirthdate {
			return p.FallbackAge < 4
		} else {
			return false
		}
	}
	age := HalfYears(p.ApproxAgeAtTime(datetime))
	return age < 4
}

// Round a duration to half-years.
func HalfYears(d time.Duration) float64 {
	return util.RoundDuration(d, Halfyear).Hours() / 24 / 365
}

func (p Person) FormattedAddressForHtml() []string {
	return strings.Split(p.Address, "\n")
}

func (p Person) GetFoodRestrictionMap() map[FoodRestriction]int {
	var restrictionMap = make(map[FoodRestriction]int)

	for _, restriction := range p.FoodRestrictions {
		restrictionMap[restriction] = 1
	}
	return restrictionMap
}

func (p Person) EncodedKey() string {
	if p.DatastoreKey == nil {
		return ""
	}
	return p.DatastoreKey.Encode()

}
