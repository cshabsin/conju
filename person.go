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
	AllFoodRestrictions      [DangerousAllergy + 1]FoodRestrictionTag
	HighlightNeededBirthdate bool
}

func makePersonUpdateFormInfo(key *datastore.Key, person Person, highlightNeededBirthdate bool) PersonUpdateFormInfo {
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

func GetAllFoodRestrictionTags() [DangerousAllergy + 1]FoodRestrictionTag {
	var toReturn [DangerousAllergy + 1]FoodRestrictionTag
	toReturn[Vegetarian] = FoodRestrictionTag{
		Tag:         Vegetarian,
		Description: "Vegetarian",
	}
	toReturn[Vegan] = FoodRestrictionTag{
		Tag:         Vegan,
		Description: "Vegan",
	}
	toReturn[NoRedMeat] = FoodRestrictionTag{
		Tag:          NoRedMeat,
		Description:  "No Red Meat",
		Supplemental: "Chicken/Fish Okay",
	}
	toReturn[VegetarianPlusFish] = FoodRestrictionTag{
		Tag:         VegetarianPlusFish,
		Description: "Vegetarian + Fish",
	}
	toReturn[NoDairy] = FoodRestrictionTag{
		Tag:         NoDairy,
		Description: "No Dairy",
	}
	toReturn[NoGluten] = FoodRestrictionTag{
		Tag:         NoGluten,
		Description: "No Gluten",
	}
	toReturn[Kosher] = FoodRestrictionTag{
		Tag:         Kosher,
		Description: "Kosher",
	}
	toReturn[Halal] = FoodRestrictionTag{
		Tag:         Halal,
		Description: "Halal",
	}
	toReturn[InconvenientAllergy] = FoodRestrictionTag{
		Tag:          InconvenientAllergy,
		Description:  "Inconvenient Allergy",
		Supplemental: "List your allergies below.",
	}
	toReturn[DangerousAllergy] = FoodRestrictionTag{
		Tag:          DangerousAllergy,
		Description:  "Dangerous Allergy",
		Supplemental: "List your allergies below.",
	}

	return toReturn
}

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
		return 0
	}
	return dateTime.Sub(p.Birthdate)
}

func (p Person) IsChildAtTime(datetime time.Time) bool {
	if p.Birthdate.IsZero() {
		return p.NeedBirthdate
	}
	age := HalfYears(p.ApproxAgeAtTime(datetime))
	return age < 16
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

	data := struct {
		People []*Person
	}{
		People: allPeople,
	}

	tpl := template.Must(template.ParseFiles("templates/test.html", "templates/listPeople.html"))
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

	formInfo := makePersonUpdateFormInfo(person.DatastoreKey, *person, false)
	infoBundle := struct {
		FormInfo PersonUpdateFormInfo
	}{
		FormInfo: formInfo,
	}
	functionMap := template.FuncMap{
		"PronounString": GetPronouns,
	}

	var tpl = template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/test.html", "templates/updatePerson.html", "templates/updatePersonForm.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "updatePerson.html", infoBundle); err != nil {
		log.Errorf(ctx, "%v", err)
	}
}

func handleSaveUpdatePerson(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	wr.Request.ParseForm()

	var p *Person
	var err error

	if wr.Request.Form.Get("EncodedKey") != "" {
		p, err = fetchPerson(wr, wr.Request.Form.Get("EncodedKey"))
		if err != nil {
			log.Errorf(ctx, "%v", err)
		}
	} else {
		newKey := PersonKey(ctx)
		p = &Person{
			NeedBirthdate: false,
		}
		p.DatastoreKey = newKey
	}

	//TODO: Is there an easier way to do this?
	//TODO: Deal with errors
	p.FirstName = wr.Request.Form.Get("FirstName")
	p.LastName = wr.Request.Form.Get("LastName")
	p.Nickname = wr.Request.Form.Get("Nickname")
	pronounConstant, e := strconv.Atoi(wr.Request.Form.Get("Pronouns"))
	if e != nil {
		pronounConstant = 0
	}
	p.Pronouns = PronounFromConstant(pronounConstant)
	p.Email = wr.Request.Form.Get("Email")
	p.Telephone = wr.Request.Form.Get("Telephone")
	p.Address = wr.Request.Form.Get("Address")
	p.Birthdate, _ = time.Parse("01/02/2006", wr.Request.Form.Get("Birthdate"))
	foodRestrictions := wr.Request.Form["FoodRestrictions"]
	var thisPersonRestrictions []FoodRestriction
	allRestrictions := GetAllFoodRestrictionTags()
	for _, restriction := range foodRestrictions {
		restrictionInt, _ := strconv.Atoi(restriction)
		thisPersonRestrictions = append(thisPersonRestrictions, allRestrictions[restrictionInt].Tag)
	}

	p.FoodRestrictions = thisPersonRestrictions
	p.FoodNotes = wr.Request.Form.Get("FoodNotes")

	p.FallbackAge, _ = strconv.ParseFloat(wr.Request.Form.Get("FallbackAge"), 64)
	p.NeedBirthdate = (wr.Request.Form.Get("NeedBirthdate") == "on")
	p.PrivateComments = wr.Request.Form.Get("PrivateComments")

	tic := time.Now()
	_, err = datastore.Put(ctx, p.DatastoreKey, p)
	if err != nil {
		log.Errorf(ctx, "%v", err)
	}

	log.Infof(ctx, "Datastore insertion took %s", time.Since(tic).String())

	// Where to go from here will depend on who's logged in and what they're doing
	http.Redirect(wr.ResponseWriter, wr.Request, "listPeople", http.StatusSeeOther)
}
