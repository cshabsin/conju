package conju

// TODO: move to "package models"?

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
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

func makePersonUpdateFormInfo(key *datastore.Key, person Person, index int, highlightNeededBirthdate bool) PersonUpdateFormInfo {
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
	return datastore.NewIncompleteKey(ctx, "Person", nil)
}

func CreatePerson(ctx context.Context, first, last string) error {
	p := Person{
		FirstName: first,
		LastName:  last,
		LoginCode: randomLoginCodeString(),
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
			return p.FallbackAge <= 5
		} else {
			return false
		}
	}
	age := HalfYears(p.ApproxAgeAtTime(datetime))
	return age <= 5
}

// Round a duration to half-years.
func HalfYears(d time.Duration) float64 {
	return RoundDuration(d, Halfyear).Hours() / 24 / 365
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
	fmt.Println(p.DatastoreKey)
	if p.DatastoreKey == nil {
		return ""
	}
	return p.DatastoreKey.Encode()

}

func handleListPeople(wr WrappedRequest) {

	ctx := appengine.NewContext(wr.Request)
	tic := time.Now()
	q := datastore.NewQuery("Person").Order("LastName").Order("FirstName")

	var allPeople []*Person
	keys, err := q.GetAll(ctx, &allPeople)
	if err != nil {
		http.Error(wr.ResponseWriter, err.Error(), http.StatusInternalServerError)
		log.Errorf(ctx, "GetAll: %v", err)
		return
	}
	log.Infof(ctx, "Datastore lookup took %s", time.Since(tic).String())
	log.Infof(ctx, "Rendering %d people", len(allPeople))

	for i := 0; i < len(allPeople); i++ {
		allPeople[i].DatastoreKey = keys[i]
	}

	wr.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := wr.MakeTemplateData(map[string]interface{}{
		"People": allPeople,
	})

	functionMap := template.FuncMap{
		"makeLoginUrl": makeLoginUrl,
	}
	tpl := template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/main.html", "templates/listPeople.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "listPeople.html", data); err != nil {
		log.Errorf(ctx, "%v", err)
	}
}

func fetchPerson(wr WrappedRequest, encodedKey string) (*Person, error) {
	ctx := appengine.NewContext(wr.Request)

	key, e := datastore.DecodeKey(encodedKey)
	if e != nil {
		log.Errorf(ctx, "%v", e)
		return nil, e
	}

	var person Person
	e = datastore.Get(ctx, key, &person)
	person.DatastoreKey = key

	if e != nil {
		log.Errorf(ctx, "%v", e)
		return nil, e
	}

	return &person, nil
}

func handleUpdatePersonForm(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)

	queryMap := wr.Request.URL.Query()

	var person *Person
	var err error

	person = &Person{
		NeedBirthdate: false,
	}

	if queryMap["key"] != nil && queryMap["key"][0] != "" {
		keyForUpdatePerson := queryMap["key"][0]
		person, err = fetchPerson(wr, keyForUpdatePerson)
		if err != nil {
			log.Errorf(ctx, "%v", err)
			http.Redirect(wr.ResponseWriter, wr.Request, "listPeople", http.StatusSeeOther)
		}
		key, _ := datastore.DecodeKey(keyForUpdatePerson)
		person.DatastoreKey = key
	}
	wr.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")

	formInfo := makePersonUpdateFormInfo(person.DatastoreKey, *person, 0, false)
	data := wr.MakeTemplateData(map[string]interface{}{
		"FormInfo": formInfo,
	})
	functionMap := template.FuncMap{
		"PronounString": GetPronouns,
	}

	var tpl = template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/main.html", "templates/updatePerson.html", "templates/updatePersonForm.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "updatePerson.html", data); err != nil {
		log.Errorf(ctx, "%v", err)
	}
}

func handleSaveUpdatePerson(wr WrappedRequest) {
	savePeople(wr)
	// Where to go from here will depend on who's logged in and what they're doing
	http.Redirect(wr.ResponseWriter, wr.Request, "listPeople", http.StatusSeeOther)
}

func savePeople(wr WrappedRequest) error {
	ctx := appengine.NewContext(wr.Request)

	wr.Request.ParseForm()
	form := wr.Request.Form

	encodedKeys := form["PersonKey"]

	for i, encodedKey := range encodedKeys {
		var p *Person
		var err error

		var key *datastore.Key
		if encodedKey != "" {
			p, err = fetchPerson(wr, encodedKey)
			if err != nil {
				log.Errorf(ctx, "%v", err)
			}
			key, err = datastore.DecodeKey(encodedKey)
		} else {
			key = PersonKey(ctx)
			p = &Person{
				NeedBirthdate: false,
				LoginCode:     randomLoginCodeString(),
			}
		}

		//TODO: Is there an easier way to do this?
		//TODO: Deal with errors
		p.FirstName = form["FirstName"][i]
		p.LastName = form["LastName"][i]
		p.Nickname = form["Nickname"][i]
		pronounConstant, e := strconv.Atoi(form["Pronouns"][i])
		if e != nil {
			pronounConstant = 0
		}
		p.Pronouns = PronounFromConstant(pronounConstant)
		p.Email = form["Email"][i]
		p.Telephone = form["Telephone"][i]
		p.Address = form["Address"][i]
		birthdate, dateError := time.Parse("01/02/2006", form["Birthdate"][i])
		if dateError == nil {
			p.Birthdate = birthdate
			if !birthdate.IsZero() && form["birthdateChanged"][i] != "0" {
				p.NeedBirthdate = false
			}

		} else {
			log.Errorf(ctx, "%v", dateError)
		}
		foodRestrictions := form[fmt.Sprintf("%s%d", "FoodRestrictions", i)]
		var thisPersonRestrictions []FoodRestriction
		allRestrictions := GetAllFoodRestrictionTags()
		for _, restriction := range foodRestrictions {
			restrictionInt, _ := strconv.Atoi(restriction)
			thisPersonRestrictions = append(thisPersonRestrictions, allRestrictions[restrictionInt].Tag)
		}

		p.FoodRestrictions = thisPersonRestrictions
		p.FoodNotes = form["FoodNotes"][i]

		if len(form["FallbackAge"]) > i {
			p.FallbackAge, _ = strconv.ParseFloat(form["FallbackAge"][i], 64)
		}
		if len(form["NeedBirthdate"]) > i {
			p.NeedBirthdate = (form["NeedBirthdate"][i] == "on")
		}
		if len(form["PrivateComments"]) > i {
			p.PrivateComments = form["PrivateComments"][i]
		}

		_, err = datastore.Put(ctx, key, p)
		if err != nil {
			log.Errorf(ctx, "------ %v", err)
		}

	}
	return nil
}
