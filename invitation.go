package conju

// TODO: move to "package models"?

import (
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

	sort.Slice(realizedInvitations,
		func(a, b int) bool {
			return SortByLastFirstName(realizedInvitations[a].Invitees[0], realizedInvitations[b].Invitees[0])
		})

	var notInvitedList []Person
	for k := range notInvitedSet {
		notInvitedList = append(notInvitedList, notInvitedSet[k])
	}
	sort.Slice(notInvitedList, func(a, b int) bool { return SortByLastFirstName(notInvitedList[a], notInvitedList[b]) })

	type EventWithKey struct {
	     Key string
	     Ev Event
	}
 
	var eventsWithKeys []EventWithKey
	if len(invitations) == 0 {
	   var allEvents []*Event
	   eventKeys, _ := datastore.NewQuery("Event").Filter("Current =", false).Order("-StartDate").GetAll(ctx, &allEvents)
	   for i := 0; i < len(eventKeys); i++ {
	       ewk := EventWithKey {
	       	   Key: eventKeys[i].Encode(),
		   Ev: *allEvents[i], 
	       }
	       eventsWithKeys = append(eventsWithKeys, ewk)
	   }
	}

	data := struct {
		CurrentEvent        Event
		Invitations         []*Invitation
		RealizedInvitations []RealizedInvitation
		NotInvitedList      []Person
		EventsWithKeys	    []EventWithKey
	}{
		CurrentEvent:        *currentEvent,
		Invitations:         invitations,
		RealizedInvitations: realizedInvitations,
		NotInvitedList:      notInvitedList,
		EventsWithKeys:      eventsWithKeys,
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
	    newInvitations = append(newInvitations, Invitation {
	         Event: currentEventKey,
	    	 Invitees: invitation.Invitees,
	    })
	    newKey := datastore.NewIncompleteKey(ctx, "Invitation", nil)
	    newInvitationKeys = append(newInvitationKeys, newKey)
	}

	datastore.PutMulti(ctx, newInvitationKeys, newInvitations)
	http.Redirect(wr.ResponseWriter, wr.Request, "invitations", http.StatusSeeOther)

}