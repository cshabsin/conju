package conju

// TODO: move to "package models"?

import (
	"context"
	"html/template"
	"net/http"
	"sort"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

type Invitation struct {
	Event    *datastore.Key   // Event
	Invitees []*datastore.Key // []Person
}

type RealizedInvitation struct {
	EncodedKey string
	Invitees   []PersonWithKey
	Event      Event
}

func makeRealizedInvitation(ctx context.Context, invitationKey datastore.Key, invitation Invitation, getEvent bool) RealizedInvitation {
	personKeys := invitation.Invitees
	var invitees []PersonWithKey
	for _, personKey := range personKeys {
		var person Person
		datastore.Get(ctx, personKey, &person)
		person.DatastoreKey = personKey
		personWithKey := PersonWithKey{
			Person: person,
			Key:    personKey.Encode(),
		}

		invitees = append(invitees, personWithKey)
	}

	var event Event

	if getEvent {
		datastore.Get(ctx, invitation.Event, &event)
	}

	realizedInvitation := RealizedInvitation{
		EncodedKey: invitationKey.Encode(),
		Invitees:   invitees,
		Event:      event,
	}

	return realizedInvitation
}

func handleInvitations(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	currentEvent := wr.Event
	currentEventKeyEncoded := wr.Values["EventKey"].(string)
	currentEventKey, _ := datastore.DecodeKey(currentEventKeyEncoded)

	var notInvitedSet = make(map[datastore.Key]PersonWithKey)
	personQuery := datastore.NewQuery("Person")
	var people []*Person
	personKeys, _ := personQuery.GetAll(ctx, &people)

	for i := 0; i < len(personKeys); i++ {
		personWithKey := PersonWithKey{Key: personKeys[i].Encode(), Person: *people[i]}
		notInvitedSet[*personKeys[i]] = personWithKey
	}

	var invitations []*Invitation

	q := datastore.NewQuery("Invitation").Filter("Event =", currentEventKey)
	var invitationKeys []*datastore.Key
	invitationKeys, _ = q.GetAll(ctx, &invitations)

	var realizedInvitations []RealizedInvitation

	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(ctx, *invitationKeys[i], *invitations[i], false)
		for _, invitee := range realizedInvitation.Invitees {
			delete(notInvitedSet, *invitee.Person.DatastoreKey)
		}

		realizedInvitations = append(realizedInvitations, realizedInvitation)
	}

	sort.Slice(realizedInvitations,
		func(a, b int) bool {
			return SortByLastFirstName(realizedInvitations[a].Invitees[0].Person, realizedInvitations[b].Invitees[0].Person)
		})

	var notInvitedList []PersonWithKey
	for k := range notInvitedSet {
		notInvitedList = append(notInvitedList, notInvitedSet[k])
	}
	sort.Slice(notInvitedList, func(a, b int) bool { return SortByLastFirstName(notInvitedList[a].Person, notInvitedList[b].Person) })

	type EventWithKey struct {
		Key string
		Ev  Event
	}

	var eventsWithKeys []EventWithKey
	if len(invitations) == 0 {
		var allEvents []*Event
		eventKeys, _ := datastore.NewQuery("Event").Filter("Current =", false).Order("-StartDate").GetAll(ctx, &allEvents)
		for i := 0; i < len(eventKeys); i++ {
			ewk := EventWithKey{
				Key: eventKeys[i].Encode(),
				Ev:  *allEvents[i],
			}
			eventsWithKeys = append(eventsWithKeys, ewk)
		}
	}

	data := struct {
		CurrentEvent        Event
		Invitations         []*Invitation
		RealizedInvitations []RealizedInvitation
		NotInvitedList      []PersonWithKey
		EventsWithKeys      []EventWithKey
	}{
		CurrentEvent:        *currentEvent,
		Invitations:         invitations,
		RealizedInvitations: realizedInvitations,
		NotInvitedList:      notInvitedList,
		EventsWithKeys:      eventsWithKeys,
	}

	functionMap := template.FuncMap{
		"ListInvitees": func(peopleWithKeys []PersonWithKey) string {
			var people []Person
			for _, person := range peopleWithKeys {
				people = append(people, person.Person)
			}
			return CollectiveAddress(people, Informal)
		},
	}

	tpl := template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/test.html", "templates/invitations.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "invitations.html", data); err != nil {
		log.Errorf(ctx, "%v", err)
	}
}

func handleCopyInvitations(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	//currentEvent := wr.Event
	currentEventKeyEncoded := wr.Values["EventKey"].(string)
	currentEventKey, _ := datastore.DecodeKey(currentEventKeyEncoded)
	wr.Request.ParseForm()

	baseEventKeyEncoded := wr.Request.Form.Get("baseEvent")
	if baseEventKeyEncoded == "" {
		return
	}

	baseEventKey, _ := datastore.DecodeKey(baseEventKeyEncoded)
	var invitations []*Invitation
	q := datastore.NewQuery("Invitation").Filter("Event =", baseEventKey)
	q.GetAll(ctx, &invitations)

	var newInvitations []Invitation
	var newInvitationKeys []*datastore.Key
	for _, invitation := range invitations {
		newInvitations = append(newInvitations, Invitation{
			Event:    currentEventKey,
			Invitees: invitation.Invitees,
		})
		newKey := datastore.NewIncompleteKey(ctx, "Invitation", nil)
		newInvitationKeys = append(newInvitationKeys, newKey)
	}

	datastore.PutMulti(ctx, newInvitationKeys, newInvitations)
	http.Redirect(wr.ResponseWriter, wr.Request, "invitations", http.StatusSeeOther)

}

func handleAddInvitation(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	currentEventKeyEncoded := wr.Values["EventKey"].(string)
	currentEventKey, _ := datastore.DecodeKey(currentEventKeyEncoded)
	wr.Request.ParseForm()

	invitationKeyEncoded := wr.Request.Form.Get("invitation")
	people := wr.Request.Form["person"]

	if len(people) == 0 {
		log.Infof(ctx, "Couldn't find any selected people!")
		return
	}

	var newPeople []*datastore.Key
	for _, person := range people {
		key, _ := datastore.DecodeKey(person)
		newPeople = append(newPeople, key)
	}

	if invitationKeyEncoded == "" {
		log.Infof(ctx, "no invitation selected, creating new one...")
		newKey := datastore.NewIncompleteKey(ctx, "Invitation", nil)
		var newInvitation Invitation
		newInvitation.Event = currentEventKey
		newInvitation.Invitees = newPeople

		_, err := datastore.Put(ctx, newKey, &newInvitation)
		if err != nil {
			log.Errorf(ctx, "%v", err)
		}
	} else {
		existingInvitationKey, _ := datastore.DecodeKey(invitationKeyEncoded)
		var existingInvitation Invitation
		datastore.Get(ctx, existingInvitationKey, &existingInvitation)
		existingInvitation.Invitees = append(existingInvitation.Invitees, newPeople...)
		_, err := datastore.Put(ctx, existingInvitationKey, &existingInvitation)
		if err != nil {
			log.Errorf(ctx, "%v", err)
		}
	}

	http.Redirect(wr.ResponseWriter, wr.Request, "invitations", http.StatusSeeOther)
}

func handleViewInvitation(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	wr.Request.ParseForm()

	invitationKeyEncoded := wr.Request.Form.Get("invitation")
	invitationKey, _ := datastore.DecodeKey(invitationKeyEncoded)
	var invitation Invitation
	datastore.Get(ctx, invitationKey, &invitation)

	data := struct {
		Invitation RealizedInvitation
	}{
		Invitation: makeRealizedInvitation(ctx, *invitationKey, invitation, true),
	}

	tpl := template.Must(template.ParseFiles("templates/test.html", "templates/viewInvitation.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "viewInvitation.html", data); err != nil {
		log.Errorf(ctx, "%v", err)
	}

}

func handleSaveInvitation(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	wr.Request.ParseForm()

	invitationKeyEncoded := wr.Request.Form.Get("invitation")
	invitationKey, _ := datastore.DecodeKey(invitationKeyEncoded)
	var invitation Invitation
	datastore.Get(ctx, invitationKey, &invitation)

	people := wr.Request.Form["person"]
	var newPeople []*datastore.Key
	for _, person := range people {
		key, _ := datastore.DecodeKey(person)
		newPeople = append(newPeople, key)
	}

	invitation.Invitees = newPeople
	_, _ = datastore.Put(ctx, invitationKey, &invitation)

	http.Redirect(wr.ResponseWriter, wr.Request, "invitations", http.StatusSeeOther)
}
