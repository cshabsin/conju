package conju

// TODO: move to "package models"?

import (
	"context"
	"html/template"
	"net/http"
	"sort"
	"strings"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

type Invitation struct {
	Event    *datastore.Key                // Event
	Invitees []*datastore.Key              // []Person
	RsvpMap  map[*datastore.Key]RsvpStatus // Person -> Rsvp
}

func (inv *Invitation) Load(ps []datastore.Property) error {
	allRsvpStatuses := GetAllRsvpStatuses()

	inv.RsvpMap = make(map[*datastore.Key]RsvpStatus)
	for _, p := range ps {
		if strings.HasPrefix(p.Name, "RsvpMap.") {
			personKey, err := datastore.DecodeKey(p.Name[8:])
			if err != nil {
				return err
			}
			rsvpInt := p.Value.(int64)
			inv.RsvpMap[personKey] = allRsvpStatuses[rsvpInt].Status
		} else if p.Name == "Event" {
			inv.Event = p.Value.(*datastore.Key)
		} else if p.Name == "Invitees" {
			inv.Invitees = append(inv.Invitees, p.Value.(*datastore.Key))
		}
	}
	return nil
}

func (inv *Invitation) Save() ([]datastore.Property, error) {
	props := []datastore.Property{
		{
			Name:  "Event",
			Value: inv.Event,
		},
	}

	for _, invitee := range inv.Invitees {
		inviteeProp := datastore.Property{
			Name:     "Invitees",
			Value:    invitee,
			Multiple: true,
		}
		props = append(props, inviteeProp)
	}

	rsvpMap := inv.RsvpMap
	for k, v := range rsvpMap {
		encodedKey := (*k).Encode()
		props = append(props, datastore.Property{Name: "RsvpMap." + encodedKey, Value: int64(v)})
	}
	return props, nil
}

type RealizedInvitation struct {
	EncodedKey string
	Invitees   []PersonWithKey
	Event      Event
	RsvpMap    map[string]RsvpStatusInfo
}

// Each event should have a list of acceptable RSVP statuses
type RsvpStatus int

const (
	No = iota
	Maybe
	FriSat
	ThuFriSat
	SatSun
	FriSatSun
	FriSatPlusEither
	WeddingOnly
	Fri
	Sat
)

type RsvpStatusInfo struct {
	Status           RsvpStatus
	ShortDescription string
	LongDescription  string
}

func GetAllRsvpStatuses() [Sat + 1]RsvpStatusInfo {
	var toReturn [Sat + 1]RsvpStatusInfo

	toReturn[No] = RsvpStatusInfo{
		Status:           No,
		ShortDescription: "No",
		LongDescription:  "Will not attend",
	}
	toReturn[Maybe] = RsvpStatusInfo{
		Status:           Maybe,
		ShortDescription: "Maybe",
		LongDescription:  "Undecided",
	}
	toReturn[FriSat] = RsvpStatusInfo{
		Status:           FriSat,
		ShortDescription: "FriSat",
		LongDescription:  "Friday - Sunday",
	}
	toReturn[ThuFriSat] = RsvpStatusInfo{
		Status:           ThuFriSat,
		ShortDescription: "ThuFriSat",
		LongDescription:  "Thursday - Sunday",
	}
	toReturn[SatSun] = RsvpStatusInfo{
		Status:           SatSun,
		ShortDescription: "SatSun",
		LongDescription:  "Saturday - Sunday",
	}
	toReturn[FriSatSun] = RsvpStatusInfo{
		Status:           FriSatSun,
		ShortDescription: "FriSatSun",
		LongDescription:  "Friday - Sunday",
	}
	toReturn[FriSatPlusEither] = RsvpStatusInfo{
		Status:           FriSatPlusEither,
		ShortDescription: "FriSatPlusEither",
		LongDescription:  "Friday - Sunday, plus either Thursday or Sunday nights",
	}
	toReturn[WeddingOnly] = RsvpStatusInfo{
		Status:           WeddingOnly,
		ShortDescription: "WeddingOnly",
		LongDescription:  "Wedding Only (no overnights)",
	}
	toReturn[Fri] = RsvpStatusInfo{
		Status:           Fri,
		ShortDescription: "Fri",
		LongDescription:  "Friday - Saturday",
	}
	toReturn[Sat] = RsvpStatusInfo{
		Status:           Sat,
		ShortDescription: "Sat",
		LongDescription:  "Saturday - Sunday",
	}
	return toReturn
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

	allRsvpStatuses := GetAllRsvpStatuses()
	realizedRsvpMap := make(map[string]RsvpStatusInfo)

	for k, v := range invitation.RsvpMap {
		realizedRsvpMap[k.Encode()] = allRsvpStatuses[v]
	}

	realizedInvitation := RealizedInvitation{
		EncodedKey: invitationKey.Encode(),
		Invitees:   invitees,
		Event:      event,
		RsvpMap:    realizedRsvpMap,
	}

	return realizedInvitation
}

func printInvitation(ctx context.Context, key datastore.Key, inv Invitation) string {
	real := makeRealizedInvitation(ctx, key, inv, true)
	toReturn := real.Event.ShortName + ": "
	for _, invitee := range real.Invitees {
		toReturn += invitee.Person.FullName() + " - "
		statusString := "???"
		status, exists := real.RsvpMap[invitee.Key]
		if exists {
			statusString = status.ShortDescription
		}
		toReturn += statusString + ", "
	}
	return toReturn
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

	t := q.Run(ctx)
	for {
		var inv Invitation
		invKey, err := t.Next(&inv)
		if err == datastore.Done {
			break
		}
		if err != nil {
			log.Errorf(ctx, "fetching next Invitation: %v", err)
			continue
		}

		log.Infof(ctx, printInvitation(ctx, *invKey, inv))
		invitationKeys = append(invitationKeys, invKey)
		invitations = append(invitations, &inv)

	}

	/*
	   	invitationKeys, err := q.GetAll(ctx, &invitations)
	               if err != nil {
	   	       log.Errorf(ctx, "Error in GetAll: %v", err)
	   	    }
	*/
	log.Infof(ctx, "Found %d invitations", len(invitations))

	var realizedInvitations []RealizedInvitation

	type Statistics struct {
		BabyCount        int
		KidCount         int
		TweenOrTeenCount int
		UnknownKidCount  int
		AdultCount       int
		UninvitedCount   int
	}

	var statistics Statistics
	statistics.UninvitedCount = len(people)
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(ctx, *invitationKeys[i], *invitations[i], false)
		for _, invitee := range realizedInvitation.Invitees {
			statistics.UninvitedCount--
			birthdate := invitee.Person.Birthdate
			if birthdate.IsZero() {
				if invitee.Person.NeedBirthdate {
					statistics.UnknownKidCount++
				} else {
					statistics.AdultCount++
				}
			} else {
				age := HalfYears(invitee.Person.ApproxAge())
				if age <= 3 {
					statistics.BabyCount++
				} else if age <= 10 {
					statistics.KidCount++
				} else if age < 16 {
					statistics.TweenOrTeenCount++
				} else {
					statistics.AdultCount++
				}
			}
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
		Stats               Statistics
	}{
		CurrentEvent:        *currentEvent,
		Invitations:         invitations,
		RealizedInvitations: realizedInvitations,
		NotInvitedList:      notInvitedList,
		EventsWithKeys:      eventsWithKeys,
		Stats:               statistics,
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

	keys, error := datastore.PutMulti(ctx, newInvitationKeys, newInvitations)
	if error != nil {
		log.Errorf(ctx, "Error in putmulti: %v", error)
	}
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

	formInfoMap := make(map[*datastore.Key]PersonUpdateFormInfo)
	realizedInvitation := makeRealizedInvitation(ctx, *invitationKey, invitation, true)
	for _, invitee := range realizedInvitation.Invitees {
		personKey := invitee.Person.DatastoreKey
		formInfo := makePersonUpdateFormInfo(personKey, invitee.Person, true, "")
		formInfoMap[personKey] = formInfo
	}

	data := struct {
		Invitation      RealizedInvitation
		FormInfoMap     map[*datastore.Key]PersonUpdateFormInfo
		AllRsvpStatuses [Sat + 1]RsvpStatusInfo
	}{
		Invitation:      realizedInvitation,
		FormInfoMap:     formInfoMap,
		AllRsvpStatuses: GetAllRsvpStatuses(),
	}

	functionMap := template.FuncMap{
		"PronounString": GetPronouns,
	}

	tpl := template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/test.html", "templates/viewInvitation.html", "templates/updatePersonForm.html"))
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
