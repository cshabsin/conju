package conju

// TODO: move to "package models"?

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/datastore"
	"google.golang.org/appengine/v2"

	"github.com/cshabsin/conju/conju/dsclient"
	"github.com/cshabsin/conju/conju/login"
	"github.com/cshabsin/conju/model/person"
)

func handleListPeople(ctx context.Context, wr WrappedRequest) {
	tic := time.Now()
	q := datastore.NewQuery("Person").Order("LastName").Order("FirstName")

	var allPeople []*person.Person
	keys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &allPeople)
	if err != nil {
		http.Error(wr.ResponseWriter, err.Error(), http.StatusInternalServerError)
		log.Printf("GetAll: %v", err)
		return
	}
	log.Printf("Datastore lookup took %s", time.Since(tic).String())
	log.Printf("Rendering %d people", len(allPeople))

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
		log.Printf("%v", err)
	}
}

func fetchPerson(wr WrappedRequest, encodedKey string) (*person.Person, error) {
	ctx := appengine.NewContext(wr.Request)

	key, e := datastore.DecodeKey(encodedKey)
	if e != nil {
		log.Printf("%v", e)
		return nil, e
	}

	var person person.Person
	e = dsclient.FromContext(ctx).Get(ctx, key, &person)
	person.DatastoreKey = key

	if e != nil {
		log.Printf("%v", e)
		return nil, e
	}

	return &person, nil
}

func handleUpdatePersonForm(ctx context.Context, wr WrappedRequest) {
	queryMap := wr.Request.URL.Query()

	var err error

	pers := &person.Person{
		NeedBirthdate: false,
	}

	if queryMap["key"] != nil && queryMap["key"][0] != "" {
		keyForUpdatePerson := queryMap["key"][0]
		pers, err = fetchPerson(wr, keyForUpdatePerson)
		if err != nil {
			log.Printf("%v", err)
			http.Redirect(wr.ResponseWriter, wr.Request, "listPeople", http.StatusSeeOther)
		}
		key, _ := datastore.DecodeKey(keyForUpdatePerson)
		pers.DatastoreKey = key
	}
	wr.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")

	formInfo := person.MakePersonUpdateFormInfo(pers.DatastoreKey, *pers, 0, false)
	data := wr.MakeTemplateData(map[string]interface{}{
		"FormInfo": formInfo,
	})
	functionMap := template.FuncMap{
		"PronounString": person.GetPronouns,
	}

	var tpl = template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/main.html", "templates/updatePerson.html", "templates/updatePersonForm.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "updatePerson.html", data); err != nil {
		log.Printf("%v", err)
	}
}

func handleSaveUpdatePerson(ctx context.Context, wr WrappedRequest) {
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
		var p *person.Person
		var err error

		var key *datastore.Key
		if encodedKey != "" {
			p, err = fetchPerson(wr, encodedKey)
			if err != nil {
				log.Printf("%v", err)
			}
			key, err = datastore.DecodeKey(encodedKey)
			if err != nil {
				log.Printf("%v", err)
			}
		} else {
			key = person.PersonKey(ctx)
			p = &person.Person{
				NeedBirthdate: false,
				LoginCode:     login.RandomLoginCodeString(),
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
		p.Pronouns = person.PronounFromConstant(pronounConstant)
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
			log.Printf("%v", dateError)
		}
		foodRestrictions := form[fmt.Sprintf("%s%d", "FoodRestrictions", i)]
		var thisPersonRestrictions []person.FoodRestriction
		allRestrictions := person.GetAllFoodRestrictionTags()
		for _, restriction := range foodRestrictions {
			restrictionInt, _ := strconv.Atoi(restriction)
			thisPersonRestrictions = append(thisPersonRestrictions, allRestrictions[restrictionInt].Tag)
		}

		p.FoodRestrictions = thisPersonRestrictions
		if form["FoodNotes"] != nil {
			p.FoodNotes = form["FoodNotes"][i]
		}

		if len(form["FallbackAge"]) > i {
			p.FallbackAge, _ = strconv.ParseFloat(form["FallbackAge"][i], 64)
		}
		if len(form["NeedBirthdate"]) > i {
			p.NeedBirthdate = (form["NeedBirthdate"][i] == "on")
		}
		if len(form["PrivateComments"]) > i {
			p.PrivateComments = form["PrivateComments"][i]
		}

		_, err = dsclient.FromContext(ctx).Put(ctx, key, p)
		if err != nil {
			log.Printf("------ %v", err)
		}

	}
	return nil
}
