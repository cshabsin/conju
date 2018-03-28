package conju

// TODO: move to "package models"?

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	//	log2 "log"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/mail"
)

type Invitation struct {
	Event                     *datastore.Key                // Event
	Invitees                  []*datastore.Key              // []Person
	RsvpMap                   map[*datastore.Key]RsvpStatus // Person -> Rsvp
	Housing                   HousingPreference
	HousingNotes              string
	HousingPreferenceBooleans int
	Driving                   DrivingPreference
	LeaveFrom                 string
	LeaveTime                 string
	AdditionalPassengers      string
	TravelNotes               string
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
		}
	}
	datastore.LoadStruct(inv, ps)
	return nil
}

func (inv *Invitation) Save() ([]datastore.Property, error) {

	x := reflect.ValueOf(*inv)

	//values := make([]interface{}, x.NumField())

	for i := 0; i < x.NumField(); i++ {
		//values[i] = x.Field(i).Interface()
		//log2.Printf("%v", x.Field(i))
	}

	//	 fmt.Println(values)

	props := []datastore.Property{
		{
			Name:  "Event",
			Value: inv.Event,
		},
		{
			Name:  "Housing",
			Value: int64(inv.Housing),
		},
		{
			Name:  "HousingNotes",
			Value: inv.HousingNotes,
		},
		{
			Name:  "HousingPreferenceBooleans",
			Value: int64(inv.HousingPreferenceBooleans),
		},
		{
			Name:  "Driving",
			Value: int64(inv.Driving),
		},
		{
			Name:  "LeaveFrom",
			Value: inv.LeaveFrom,
		},
		{
			Name:  "LeaveTime",
			Value: inv.LeaveTime,
		},
		{
			Name:  "AdditionalPassengers",
			Value: inv.AdditionalPassengers,
		},
		{
			Name:  "TravelNotes",
			Value: inv.TravelNotes,
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

func (inv *Invitation) HasChildren(ctx context.Context) bool {
	var event Event
	datastore.Get(ctx, inv.Event, &event)
	for _, personKey := range inv.Invitees {
		var person Person
		datastore.Get(ctx, personKey, &person)
		if person.IsChildAtTime(event.StartDate) {
			return true
		}
	}
	return false
}

func (inv *Invitation) HasHousingPreference(preference HousingPreferenceBoolean) bool {
	return (inv.HousingPreferenceBooleans & int(preference)) > 0
}

type RealizedInvitation struct {
	EncodedKey                string
	Invitees                  []PersonWithKey
	Event                     Event
	RsvpMap                   map[string]RsvpStatusInfo
	Housing                   HousingPreference
	HousingPreferenceBooleans int
	HousingNotes              string
	Driving                   DrivingPreference
	LeaveFrom                 string
	LeaveTime                 string
	TravelNotes               string
	AdditionalPassengers      string
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
		EncodedKey:                invitationKey.Encode(),
		Invitees:                  invitees,
		Event:                     event,
		RsvpMap:                   realizedRsvpMap,
		Housing:                   invitation.Housing,
		HousingNotes:              invitation.HousingNotes,
		HousingPreferenceBooleans: invitation.HousingPreferenceBooleans,
		Driving:                   invitation.Driving,
		LeaveFrom:                 invitation.LeaveFrom,
		LeaveTime:                 invitation.LeaveTime,
		AdditionalPassengers:      invitation.AdditionalPassengers,
		TravelNotes:               invitation.TravelNotes,
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

		invitationKeys = append(invitationKeys, invKey)
		invitations = append(invitations, &inv)

	}

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
				age := HalfYears(invitee.Person.ApproxAgeAtTime(wr.Event.StartDate))
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

	_, error := datastore.PutMulti(ctx, newInvitationKeys, newInvitations)
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
	for i, invitee := range realizedInvitation.Invitees {
		personKey := invitee.Person.DatastoreKey
		formInfo := makePersonUpdateFormInfo(personKey, invitee.Person, i, true)
		formInfoMap[personKey] = formInfo
	}

	data := struct {
		Invitation                   RealizedInvitation
		FormInfoMap                  map[*datastore.Key]PersonUpdateFormInfo
		AllRsvpStatuses              []RsvpStatusInfo
		AllHousingPreferences        []HousingPreferenceInfo
		AllHousingPreferenceBooleans []HousingPreferenceBooleanInfo
		AllDrivingPreferences        []DrivingPreferenceInfo
		InvitationHasChildren        bool
	}{
		Invitation:                   realizedInvitation,
		FormInfoMap:                  formInfoMap,
		AllRsvpStatuses:              GetAllRsvpStatuses(),
		AllHousingPreferences:        GetAllHousingPreferences(),
		AllHousingPreferenceBooleans: GetAllHousingPreferenceBooleans(),
		AllDrivingPreferences:        GetAllDrivingPreferences(),
		InvitationHasChildren:        invitation.HasChildren(ctx),
	}

	functionMap := template.FuncMap{
		"PronounString": GetPronouns,
		"HasPreference": HasPreference,
	}

	tpl := template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/test.html", "templates/viewInvitation.html", "templates/updatePersonForm.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "viewInvitation.html", data); err != nil {
		log.Errorf(ctx, "%v", err)
	}

}

func HasPreference(total int, mask int) bool {
	return (total & mask) != 0
}

func handleSaveInvitation(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	wr.Request.ParseForm()

	emailBody := ""

	invitationKeyEncoded := wr.Request.Form.Get("invitation")
	invitationKey, _ := datastore.DecodeKey(invitationKeyEncoded)
	var invitation Invitation
	datastore.Get(ctx, invitationKey, &invitation)

	people := wr.Request.Form["person"]
	rsvps := wr.Request.Form["rsvp"]
	var newPeople []*datastore.Key
	var rsvpMap = make(map[*datastore.Key]RsvpStatus)
	for i, personKey := range people {
		key, _ := datastore.DecodeKey(personKey)
		var person Person
		datastore.Get(ctx, key, &person)
		newPeople = append(newPeople, key)
		rsvp, _ := strconv.Atoi(rsvps[i])
		rsvpStatusString := "No RSVP"
		if rsvp >= 0 {
			fullStatus := GetAllRsvpStatuses()[rsvp]
			rsvpMap[key] = fullStatus.Status
			rsvpStatusString = fullStatus.ShortDescription
		}

		emailBody += fmt.Sprintf("%s: %s\n", person.FullName(), rsvpStatusString)
	}

	emailBody += "\n\n"
	invitation.RsvpMap = rsvpMap

	invitation.Invitees = newPeople

	housingPreference, _ := strconv.Atoi(wr.Request.Form.Get("housingPreference"))
	if housingPreference >= 0 {
		hp := HousingPreference(housingPreference)
		invitation.Housing = hp
		emailBody += fmt.Sprintf("Housing Preference: %s\n ", GetAllHousingPreferences()[hp].ReportDescription)
	}

	var booleanInfos = GetAllHousingPreferenceBooleans()
	var housingPreferenceTotal int
	booleans := wr.Request.Form["housingPreferenceBooleans"]

	if len(booleans) > 0 {
		emailBody += "Housing Flags: "
	}
	for _, boolean := range booleans {
		value, _ := strconv.Atoi(boolean)
		booleanInfo := booleanInfos[value]
		housingPreferenceTotal += booleanInfo.Bit

		emailBody += booleanInfo.ReportDescription + ", "
	}
	if len(booleans) > 0 {
		emailBody += "\n"
	}

	invitation.HousingPreferenceBooleans = housingPreferenceTotal

	invitation.HousingNotes = wr.Request.Form.Get("housingNotes")
	if invitation.HousingNotes != "" {
		emailBody += "Housing Notes: " + invitation.HousingNotes + "\n\n"
	}

	drivingPreference, _ := strconv.Atoi(wr.Request.Form.Get("drivingPreference"))
	invitation.Driving = DrivingPreference(drivingPreference)

	invitation.LeaveFrom = wr.Request.Form.Get("leaveFrom")
	invitation.LeaveTime = wr.Request.Form.Get("leaveTime")
	invitation.AdditionalPassengers = wr.Request.Form.Get("additionalPassengers")
	invitation.TravelNotes = wr.Request.Form.Get("travelNotes")

	_, err := datastore.Put(ctx, invitationKey, &invitation)
	if err != nil {
		log.Errorf(ctx, "%v", err)
	}

	var invitees []Person
	for _, personKey := range invitation.Invitees {
		var person Person
		datastore.Get(ctx, personKey, &person)
		invitees = append(invitees, person)
	}

	var e Event
	datastore.Get(ctx, invitation.Event, &e)
	subject := fmt.Sprintf("%s: RSVP from %s", e.ShortName, CollectiveAddress(invitees, Informal))

	// TODO: figure out how to set these with a properties file.
	msg := &mail.Message{
		Sender:  "**** email sender ****",
		To:      []string{"**** email address ****"},
		Subject: subject,
		Body:    emailBody,
	}
	if err := mail.Send(ctx, msg); err != nil {
		log.Errorf(ctx, "Couldn't send email: %v", err)
	}

	savePeople(wr)

	http.Redirect(wr.ResponseWriter, wr.Request, "invitations", http.StatusSeeOther)
}
