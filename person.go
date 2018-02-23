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
	DatastoreKey *datastore.Key
	FirstName    string
	LastName     string
	Nickname     string
	Pronouns     int
	Email        string
	Telephone    string
	Address      string
	Birthdate    time.Time
	IsAdmin      bool
	FallbackAge  float64
	//TODO: make this nilable
	NeedBirthdate bool
	// these fields can be removed after all the data is ported
	OldGuestId    int
	OldInviteeId  int
	OldInviteCode string
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

func (p Person) FormattedAddressForHtml() []string {
	return strings.Split(p.Address, "\n")
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
		return nil, e
	}
	q := datastore.NewQuery("Person").Filter("__key__ =", key)

	//TODO: alternatives to GetAll
	var p []*Person

	tic := time.Now()
	keys, e := q.GetAll(ctx, &p)
	log.Infof(ctx, "Datastore lookup took %s", time.Since(tic).String())
	if e != nil {
		return nil, e
	}

	person := p[0]
	person.DatastoreKey = keys[0]

	return person, nil
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
	}
	wr.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := struct {
		ThisPerson *Person
	}{
		ThisPerson: person,
	}

	var tpl = template.Must(template.ParseFiles("templates/test.html", "templates/updatePerson.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "updatePerson.html", data); err != nil {
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
	p.Pronouns, _ = strconv.Atoi(wr.Request.Form.Get("Pronouns"))
	p.Email = wr.Request.Form.Get("Email")
	p.Telephone = wr.Request.Form.Get("Telephone")
	p.Address = wr.Request.Form.Get("Address")
	p.Birthdate, _ = time.Parse("01/02/2006", wr.Request.Form.Get("Birthdate"))
	p.FallbackAge, _ = strconv.ParseFloat(wr.Request.Form.Get("FallbackAge"), 64)
	p.NeedBirthdate = (wr.Request.Form.Get("NeedBirthdate") == "on")

	tic := time.Now()
	_, err = datastore.Put(ctx, p.DatastoreKey, p)
	if err != nil {
		log.Errorf(ctx, "%v", err)
	}

	log.Infof(ctx, "Datastore insertion took %s", time.Since(tic).String())

	// Where to go from here will depend on who's logged in and what they're doing
	http.Redirect(wr.ResponseWriter, wr.Request, "listPeople", http.StatusSeeOther)
}
