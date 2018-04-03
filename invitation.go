package conju

// TODO: move to "package models"?

import (
	"context"
	"fmt"
	//	"html"
	"html/template"
	log2 "log"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

type Invitation struct {
	Event                     *datastore.Key                // Event
	Invitees                  []*datastore.Key              // []Person
	RsvpMap                   map[*datastore.Key]RsvpStatus // Person -> Rsvp
	ActivityMap               map[*datastore.Key](map[*datastore.Key]ActivityRanking)
	ActivityLeaderMap         map[*datastore.Key](map[*datastore.Key]bool)
	Housing                   HousingPreference
	HousingNotes              string
	HousingPreferenceBooleans int
	Driving                   DrivingPreference
	Parking                   ParkingType
	LeaveFrom                 string
	LeaveTime                 string
	AdditionalPassengers      string
	TravelNotes               string
	OtherInfo                 string
}

func (inv *Invitation) Load(ps []datastore.Property) error {
	allRsvpStatuses := GetAllRsvpStatuses()

	inv.RsvpMap = make(map[*datastore.Key]RsvpStatus)
	inv.ActivityMap = make(map[*datastore.Key](map[*datastore.Key]ActivityRanking))
	inv.ActivityLeaderMap = make(map[*datastore.Key](map[*datastore.Key]bool))
	for _, p := range ps {
		if strings.HasPrefix(p.Name, "RsvpMap.") {
			personKey, err := datastore.DecodeKey(p.Name[8:])
			if err != nil {
				return err
			}
			rsvpInt := p.Value.(int64)
			inv.RsvpMap[personKey] = allRsvpStatuses[rsvpInt].Status
		}

		if strings.HasPrefix(p.Name, "ActivityMap.") {

			underscore := strings.Index(p.Name, "_")
			personKeyString := p.Name[12:underscore]
			personKey, err := datastore.DecodeKey(personKeyString)
			if err != nil {
				log2.Printf("person lookup error: %v", err)
			}

			mapForPerson := make(map[*datastore.Key]ActivityRanking)
			// Ewwwwww
			for person, m := range inv.ActivityMap {
				if *person == *personKey {
					mapForPerson = m
					break
				}
			}

			inv.ActivityMap[personKey] = mapForPerson

			activityKeyString := p.Name[underscore+1:]

			activityKey, err := datastore.DecodeKey(activityKeyString)
			if err != nil {
				return err
			}

			activityRankingInt := p.Value.(int64)
			mapForPerson[activityKey] = ActivityRanking(activityRankingInt)
		}

		if strings.HasPrefix(p.Name, "ActivityLeaderMap.") {

			underscore := strings.Index(p.Name, "_")
			personKeyString := p.Name[18:underscore]
			personKey, err := datastore.DecodeKey(personKeyString)
			if err != nil {
				log2.Printf("person lookup error: %v", err)
			}

			mapForPerson := make(map[*datastore.Key]bool)
			// Ewwwwww
			for person, m := range inv.ActivityLeaderMap {
				if *person == *personKey {
					mapForPerson = m
					break
				}
			}

			inv.ActivityLeaderMap[personKey] = mapForPerson

			activityKeyString := p.Name[underscore+1:]

			activityKey, err := datastore.DecodeKey(activityKeyString)
			if err != nil {
				return err
			}

			mapForPerson[activityKey] = p.Value.(bool)
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
			Name:  "Parking",
			Value: int64(inv.Parking),
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
		{Name: "OtherInfo",
			Value: inv.OtherInfo,
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

	activityMap := inv.ActivityMap
	for p, m := range activityMap {
		personEncodedKey := (*p).Encode()
		partialName := "ActivityMap." + personEncodedKey
		for a, v := range m {
			totalKey := partialName + "_" + (*a).Encode()
			props = append(props, datastore.Property{Name: totalKey, Value: int64(v)})
		}
	}
	activityLeaderMap := inv.ActivityLeaderMap
	for p, m := range activityLeaderMap {
		personEncodedKey := (*p).Encode()
		partialName := "ActivityLeaderMap." + personEncodedKey
		for a, v := range m {
			totalKey := partialName + "_" + (*a).Encode()
			props = append(props, datastore.Property{Name: totalKey, Value: v})
		}
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
	return (inv.HousingPreferenceBooleans & GetAllHousingPreferenceBooleans()[preference].Bit) > 0
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

	realizedInvitations := makeRealizedInvitations(ctx, invitationKeys, invitations)

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
		realizedInvitation := realizedInvitations[i]
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
		eventKeys, err := datastore.NewQuery("Event").Filter("Current =", false).Order("-StartDate").GetAll(ctx, &allEvents)
		if err != nil {
			log.Errorf(ctx, "Error listing events for copyInvitations: %v", err)
		}
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

	tpl := template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/main.html", "templates/invitations.html"))
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

// handleViewInvitationUser handles /viewInvitation URLs.
func handleViewInvitationAdmin(wr WrappedRequest) {
	wr.Request.ParseForm()

	invitationKeyEncoded := wr.Request.Form.Get("invitation")
	invitationKey, err := datastore.DecodeKey(invitationKeyEncoded)
	if err != nil {
		http.Error(wr.ResponseWriter,
			fmt.Sprintf("Error decoding invitation key: %v", err),
			http.StatusBadRequest)
	}
	handleViewInvitation(wr, invitationKey)
}

// handleViewInvitationUser handles /rsvp URLs.
func handleViewInvitationUser(wr WrappedRequest) {
	handleViewInvitation(wr, wr.InvitationKey)
}

func handleViewInvitation(wr WrappedRequest, invitationKey *datastore.Key) {
	var invitation Invitation
	datastore.Get(wr.Context, invitationKey, &invitation)

	formInfoMap := make(map[*datastore.Key]PersonUpdateFormInfo)
	realizedInvitation := makeRealizedInvitation(wr.Context, *invitationKey, invitation)
	for i, invitee := range realizedInvitation.Invitees {
		personKey := invitee.Person.DatastoreKey
		formInfo := makePersonUpdateFormInfo(personKey, invitee.Person, i, true)
		formInfoMap[personKey] = formInfo
	}

	activityKeys := realizedInvitation.Event.Activities
	var activities = make([]*Activity, len(activityKeys))
	err := datastore.GetMulti(wr.Context, activityKeys, activities)
	if err != nil {
		log.Infof(wr.Context, "%v", err)
	}

	var realActivities []Activity
	for _, activity := range activities {
		realActivities = append(realActivities, *activity)
	}

	data := struct {
		CurrentEvent                 Event
		Invitation                   RealizedInvitation
		FormInfoMap                  map[*datastore.Key]PersonUpdateFormInfo
		AllRsvpStatuses              []RsvpStatusInfo
		Activities                   []Activity
		AllHousingPreferences        []HousingPreferenceInfo
		AllHousingPreferenceBooleans []HousingPreferenceBooleanInfo
		AllDrivingPreferences        []DrivingPreferenceInfo
		AllParkingTypes              []ParkingTypeInfo
		InvitationHasChildren        bool
	}{
		CurrentEvent:                 *wr.Event,
		Invitation:                   realizedInvitation,
		FormInfoMap:                  formInfoMap,
		AllRsvpStatuses:              GetAllRsvpStatuses(),
		Activities:                   realActivities,
		AllHousingPreferences:        GetAllHousingPreferences(),
		AllHousingPreferenceBooleans: GetAllHousingPreferenceBooleans(),
		AllDrivingPreferences:        GetAllDrivingPreferences(),
		AllParkingTypes:              GetAllParkingTypes(),
		InvitationHasChildren:        invitation.HasChildren(wr.Context),
	}

	functionMap := template.FuncMap{
		"PronounString": GetPronouns,
		"HasPreference": HasPreference,
	}

	tpl := template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/main.html", "templates/viewInvitation.html", "templates/updatePersonForm.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "viewInvitation.html", data); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}

func HasPreference(total int, mask int) bool {
	return (total & mask) != 0
}

func handleSaveInvitation(wr WrappedRequest) {
	wr.Request.ParseForm()

	invitationKeyEncoded := wr.Request.Form.Get("invitation")
	invitationKey, _ := datastore.DecodeKey(invitationKeyEncoded)
	var invitation Invitation
	datastore.Get(wr.Context, invitationKey, &invitation)

	people := wr.Request.Form["person"]
	rsvps := wr.Request.Form["rsvp"]
	var newPeople []*datastore.Key
	var rsvpMap = make(map[*datastore.Key]RsvpStatus)
	var activityMap = make(map[*datastore.Key](map[*datastore.Key]ActivityRanking))
	var activityLeaderMap = make(map[*datastore.Key](map[*datastore.Key]bool))
	for i, personKey := range people {
		key, _ := datastore.DecodeKey(personKey)
		var person Person
		datastore.Get(wr.Context, key, &person)
		newPeople = append(newPeople, key)
		rsvp, _ := strconv.Atoi(rsvps[i])
		if rsvp >= 0 {
			fullStatus := GetAllRsvpStatuses()[rsvp]
			rsvpMap[key] = fullStatus.Status
		}

		var activityMapForPerson = make(map[*datastore.Key]ActivityRanking)
		var activityLeaderMapForPerson = make(map[*datastore.Key]bool)
		for a, activityKey := range wr.Event.Activities {
			ranking := wr.Request.Form.Get(strings.Join([]string{"activity_", strconv.Itoa(i), "_", strconv.Itoa(a)}, ""))
			rankingInt, _ := strconv.Atoi(ranking)
			activityMapForPerson[activityKey] = ActivityRanking(rankingInt)
			leader := wr.Request.Form.Get(strings.Join([]string{"activity_", strconv.Itoa(i), "_", strconv.Itoa(a), "_leader"}, ""))
			if leader == "on" {
				activityLeaderMapForPerson[activityKey] = true
			}
		}
		activityMap[key] = activityMapForPerson
		activityLeaderMap[key] = activityLeaderMapForPerson
	}

	invitation.RsvpMap = rsvpMap
	invitation.ActivityMap = activityMap
	invitation.ActivityLeaderMap = activityLeaderMap

	invitation.Invitees = newPeople

	housingPreference, _ := strconv.Atoi(wr.Request.Form.Get("housingPreference"))
	if housingPreference >= 0 {
		hp := HousingPreference(housingPreference)
		invitation.Housing = hp
	}

	var booleanInfos = GetAllHousingPreferenceBooleans()
	var housingPreferenceTotal int
	booleans := wr.Request.Form["housingPreferenceBooleans"]

	for _, boolean := range booleans {
		value, _ := strconv.Atoi(boolean)
		booleanInfo := booleanInfos[value]
		housingPreferenceTotal += booleanInfo.Bit
	}

	invitation.HousingPreferenceBooleans = housingPreferenceTotal

	invitation.HousingNotes = wr.Request.Form.Get("housingNotes")

	drivingPreference, _ := strconv.Atoi(wr.Request.Form.Get("drivingPreference"))
	invitation.Driving = DrivingPreference(drivingPreference)

	parkingType, _ := strconv.Atoi(wr.Request.Form.Get("parking"))
	if parkingType >= 0 {
		pt := ParkingType(parkingType)
		invitation.Parking = pt
	}

	invitation.LeaveFrom = wr.Request.Form.Get("leaveFrom")
	invitation.LeaveTime = wr.Request.Form.Get("leaveTime")
	invitation.AdditionalPassengers = wr.Request.Form.Get("additionalPassengers")
	invitation.TravelNotes = wr.Request.Form.Get("travelNotes")
	invitation.OtherInfo = wr.Request.Form.Get("otherInfo")

	_, err := datastore.Put(wr.Context, invitationKey, &invitation)
	if err != nil {
		log.Errorf(wr.Context, "%v", err)
	}

	var invitees []Person
	for _, personKey := range invitation.Invitees {
		var person Person
		datastore.Get(wr.Context, personKey, &person)
		invitees = append(invitees, person)
	}

	savePeople(wr)

	log.Infof(wr.Context, "-------%v", wr.AdminInfo)
	if !wr.User.Admin {

		data := struct {
			AnyAttending bool
		}{
			AnyAttending: true,
		}

		tpl := template.Must(template.ParseFiles("templates/main.html", "templates/thanks.html"))
		if err := tpl.ExecuteTemplate(wr.ResponseWriter, "viewInvitation.html", data); err != nil {
			log.Errorf(wr.Context, "%v", err)
		}

		return
	}

	type NewPersonInfo struct {
		Name        string
		Description string
	}
	newPeopleNames := wr.Request.Form["newPersonName"]
	newPeopleDescs := wr.Request.Form["newPersonDescription"]

	var additionalPeople []NewPersonInfo
	for i, name := range newPeopleNames {
		additionalPeople = append(additionalPeople, NewPersonInfo{Name: name, Description: newPeopleDescs[i]})
	}

	newPeopleSubjectFragment := ""
	if len(additionalPeople) > 0 {
		newPeopleSubjectFragment = " ADDITION REQUESTED,"
	}

	anyAttending := false
	var isAttending []bool
	for _, invitee := range invitation.Invitees {
		if rsvp, present := rsvpMap[invitee]; present {
			attending := GetAllRsvpStatuses()[rsvp].Attending
			isAttending = append(isAttending, attending)
			anyAttending = anyAttending || attending
		} else {
			isAttending = append(isAttending, false)
		}
	}

	var e Event
	datastore.Get(wr.Context, invitation.Event, &e)
	subject := fmt.Sprintf("%s:%s RSVP from %s", e.ShortName, newPeopleSubjectFragment, CollectiveAddress(invitees, Informal))

	functionMap := template.FuncMap{
		"HasHousingPreference": RealInvHasHousingPreference,
		"PronounString":        GetPronouns,
	}

	realizedInvitation := makeRealizedInvitation(wr.Context, *invitationKey, invitation)
	// TODO: escape this.
	//realizedInvitation.HousingNotes = strings.Replace(realizedInvitation.HousingNotes, "\n", "<br>", -1)

	log.Infof(wr.Context, "isAttending: %v, anyAttending: %v", isAttending, anyAttending)

	data := struct {
		RealInvitation               RealizedInvitation
		AllHousingPreferenceBooleans []HousingPreferenceBooleanInfo
		AllPronouns                  []PronounSet
		AllFoodRestrictions          []FoodRestrictionTag
		AdditionalPeople             []NewPersonInfo
		AnyAttending                 bool
		IsAttending                  []bool
	}{
		RealInvitation:               realizedInvitation,
		AllHousingPreferenceBooleans: GetAllHousingPreferenceBooleans(),
		AllPronouns:                  []PronounSet{They, She, He, Zie},
		AllFoodRestrictions:          GetAllFoodRestrictionTags(),
		AdditionalPeople:             additionalPeople,
		AnyAttending:                 anyAttending,
		IsAttending:                  isAttending,
	}

	header := MailHeaderInfo{
		To:      []string{"**** email address ****"},
		Subject: subject,
	}
	sendMail(wr.Context, "rsvpconfirmation", data, functionMap, header)
	http.Redirect(wr.ResponseWriter, wr.Request, "invitations", http.StatusSeeOther)
}
