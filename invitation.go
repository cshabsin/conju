package conju

// TODO: move to "package models"?

import (
	"html/template"
	"sort"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

type Invitation struct {
	Event    *datastore.Key   // Event
	Invitees []*datastore.Key // []Person
}

func handleInvitations(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	currentEvent := wr.Event
	currentEventKeyEncoded := wr.Values["EventKey"].(string)
	currentEventKey, _ := datastore.DecodeKey(currentEventKeyEncoded)

	var notInvitedSet = make(map[datastore.Key]Person)
	personQuery := datastore.NewQuery("Person")
	var people []*Person
	personKeys, _ := personQuery.GetAll(ctx, &people)

	for i := 0; i < len(personKeys); i++ {
		notInvitedSet[*personKeys[i]] = *people[i]
	}

	var invitations []*Invitation

	q := datastore.NewQuery("Invitation").Filter("Event =", currentEventKey)
	var invitationKeys []*datastore.Key
	invitationKeys, _ = q.GetAll(ctx, &invitations)

	type RealizedInvitation struct {
		EncodedKey string
		Invitees   []Person
	}

	var realizedInvitations []RealizedInvitation

	for i := 0; i < len(invitations); i++ {

		personKeys := invitations[i].Invitees
		var invitees []Person
		for _, personKey := range personKeys {
			var person Person
			datastore.Get(ctx, personKey, &person)
			invitees = append(invitees, person)
			delete(notInvitedSet, *personKey)
		}

		realizedInvitation := RealizedInvitation{
			EncodedKey: invitationKeys[i].Encode(),
			Invitees:   invitees,
		}

		realizedInvitations = append(realizedInvitations, realizedInvitation)
	}

	var notInvitedList []Person
	for k := range notInvitedSet {
		notInvitedList = append(notInvitedList, notInvitedSet[k])
	}
	sort.Slice(notInvitedList, func(a, b int) bool { return SortByLastFirstName(notInvitedList[a], notInvitedList[b]) })

	data := struct {
		CurrentEvent        Event
		Invitations         []*Invitation
		RealizedInvitations []RealizedInvitation
		NotInvitedList      []Person
	}{
		CurrentEvent:        *currentEvent,
		Invitations:         invitations,
		RealizedInvitations: realizedInvitations,
		NotInvitedList:      notInvitedList,
	}

	functionMap := template.FuncMap{
		"ListInvitees": func(people []Person) string {
			return CollectiveAddress(people, Informal)
		},
	}

	tpl := template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/test.html", "templates/invitations.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "invitations.html", data); err != nil {
		log.Errorf(ctx, "%v", err)
	}
}
