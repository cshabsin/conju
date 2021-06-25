package conju

// TODO: move to "package models"?

import (
	"context"
	"fmt"
	"html/template"
	log2 "log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cshabsin/conju/activity"
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
	ThursdayDinnerCount       int
	FridayLunch               bool
	FridayDinnerCount         int
	FridayIceCreamCount       int
	OtherInfo                 string
	LastUpdatedPerson         *datastore.Key
	LastUpdatedTimestamp      time.Time
	ReceivedPay               float64 // US Dollars
	ReceivedPayMethod         string
	ReceivedPayDate           time.Time
}

const delimiter = "|_|"

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

			delimiterIndex := strings.Index(p.Name, delimiter)
			personKeyString := p.Name[12:delimiterIndex]
			personKey, err := datastore.DecodeKey(personKeyString)
			if err != nil {
				log2.Printf("person lookup error: %v", err)
				continue
			}
			if personKey == nil {
				log2.Printf("person lookup yielded nil key for map entry %s", p.Name)
				continue
			}

			mapForPerson := make(map[*datastore.Key]ActivityRanking)
			// Ewwwwww
			var mainPersonKey *datastore.Key
			for person, m := range inv.ActivityMap {
				if *person == *personKey {
					mapForPerson = m
					mainPersonKey = person
					break
				}
			}
			if mainPersonKey == nil {
				mainPersonKey = personKey
			}

			inv.ActivityMap[mainPersonKey] = mapForPerson

			activityKeyString := p.Name[(delimiterIndex + len(delimiter)):]

			activityKey, err := datastore.DecodeKey(activityKeyString)
			if err != nil {
				return err
			}

			activityRankingInt := p.Value.(int64)
			mapForPerson[activityKey] = ActivityRanking(activityRankingInt)
		}

		if strings.HasPrefix(p.Name, "ActivityLeaderMap.") {

			delimiterIndex := strings.Index(p.Name, delimiter)
			personKeyString := p.Name[18:delimiterIndex]
			personKey, err := datastore.DecodeKey(personKeyString)
			if err != nil {
				log2.Printf("person lookup error: %v", err)
			}

			mapForPerson := make(map[*datastore.Key]bool)
			// Ewwwwww
			var mainPersonKey *datastore.Key
			for person, m := range inv.ActivityLeaderMap {
				if *person == *personKey {
					mapForPerson = m
					mainPersonKey = person
					break
				}
			}
			if mainPersonKey == nil {
				mainPersonKey = personKey
			}

			inv.ActivityLeaderMap[mainPersonKey] = mapForPerson

			activityKeyString := p.Name[(delimiterIndex + len(delimiter)):]

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
		{
			Name:  "ThursdayDinnerCount",
			Value: int64(inv.ThursdayDinnerCount),
		},
		{
			Name:  "FridayLunch",
			Value: inv.FridayLunch,
		},
		{
			Name:  "FridayDinnerCount",
			Value: int64(inv.FridayDinnerCount),
		},
		{
			Name:  "FridayIceCreamCount",
			Value: int64(inv.FridayIceCreamCount),
		},
		{
			Name:  "OtherInfo",
			Value: inv.OtherInfo,
		},
		{
			Name:  "LastUpdatedPerson",
			Value: inv.LastUpdatedPerson,
		},
		{
			Name:  "LastUpdatedTimestamp",
			Value: inv.LastUpdatedTimestamp,
		},
		{
			Name:  "ReceivedPay",
			Value: float64(inv.ReceivedPay),
		},
		{
			Name:  "ReceivedPayDate",
			Value: inv.ReceivedPayDate,
		},
		{
			Name:  "ReceivedPayMethod",
			Value: inv.ReceivedPayMethod,
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
			totalKey := partialName + delimiter + (*a).Encode()
			props = append(props, datastore.Property{Name: totalKey, Value: int64(v)})
		}
	}
	activityLeaderMap := inv.ActivityLeaderMap
	for p, m := range activityLeaderMap {
		personEncodedKey := (*p).Encode()
		partialName := "ActivityLeaderMap." + personEncodedKey
		for a, v := range m {
			totalKey := partialName + delimiter + (*a).Encode()
			props = append(props, datastore.Property{Name: totalKey, Value: v})
		}
	}

	return props, nil
}

func (inv *Invitation) AnyAttending() bool {
	allStatuses := GetAllRsvpStatuses()
	for _, v := range inv.RsvpMap {
		attending := allStatuses[v].Attending
		if attending {
			return attending
		}
	}
	return false
}

func (inv *Invitation) AttendingInvitees() []*datastore.Key {
	allStatuses := GetAllRsvpStatuses()
	var attending []*datastore.Key
	for k, v := range inv.RsvpMap {
		if allStatuses[v].Attending {
			attending = append(attending, k)
		}
	}
	return attending
}

func (inv *Invitation) AnyUndecided() bool {
	allStatuses := GetAllRsvpStatuses()
	for _, invitee := range inv.Invitees {
		if rsvp, present := inv.RsvpMap[invitee]; present {
			undecided := allStatuses[rsvp].Undecided
			if undecided {
				return true
			}
		} else {
			return true
		}
	}
	return false
}

func (inv *Invitation) HasChildren(ctx context.Context) bool {
	var event Event
	datastore.Get(ctx, inv.Event, &event)
	for _, personKey := range inv.Invitees {
		var person Person
		datastore.Get(ctx, personKey, &person)
		if person.IsNonAdultAtTime(event.StartDate) {
			return true
		}
	}
	return false
}

func (inv *Invitation) HasHousingPreference(preference HousingPreferenceBoolean) bool {
	return (inv.HousingPreferenceBooleans & GetAllHousingPreferenceBooleans()[preference].Bit) > 0
}

// Handles /invitations, listing invitations.
func handleInvitations(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	currentEventKey := wr.EventKey

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
	invitationKeys, err := q.GetAll(ctx, &invitations)
	if err != nil {
		log.Errorf(ctx, "fetching invitations: %v", err)
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
	for _, realizedInvitation := range realizedInvitations {
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

	notInvitedList := make([]PersonWithKey, len(notInvitedSet), len(notInvitedSet))
	i := 0
	for k := range notInvitedSet {
		notInvitedList[i] = notInvitedSet[k]
		i++
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

	data := wr.MakeTemplateData(map[string]interface{}{
		"Invitations":         invitations,
		"RealizedInvitations": realizedInvitations,
		"NotInvitedList":      notInvitedList,
		"EventsWithKeys":      eventsWithKeys,
		"Stats":               statistics,
	})

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
	currentEventKey := wr.EventKey
	wr.Request.ParseForm()

	baseEventKeyEncoded := wr.Request.Form.Get("baseEvent")
	log.Infof(ctx, "Found base event: %v", baseEventKeyEncoded)
	if baseEventKeyEncoded == "" {
		return
	}

	baseEventKey, err := datastore.DecodeKey(baseEventKeyEncoded)
	log.Infof(ctx, "error decoding event key: %v", err)
	var invitations []*Invitation
	q := datastore.NewQuery("Invitation").Filter("Event =", baseEventKey)
	q.GetAll(ctx, &invitations)

	log.Infof(ctx, "Found %d invitations from copied event", len(invitations))
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
	currentEventKey := wr.EventKey
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

func handleDeleteInvitation(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	wr.Request.ParseForm()

	invitationKeyEncoded := wr.Request.Form.Get("invitation")
	invitationKey, err := datastore.DecodeKey(invitationKeyEncoded)
	if err != nil {
		log.Errorf(ctx, "key decryption error: %v", err)
	}

	err = datastore.Delete(ctx, invitationKey)
	if err != nil {
		log.Errorf(ctx, "invitation deletion error: %v", err)
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

var (
	functionMap = template.FuncMap{
		"PronounString":               GetPronouns,
		"HasPreference":               HasPreference,
		"DerefPeople":                 DerefPeople,
		"CollectiveAddressFirstNames": CollectiveAddressFirstNames,
		"SharerName":                  MakeSharerName,
	}

	invitationTpl = template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/main.html", "templates/viewInvitation.html", "templates/updatePersonForm.html", "templates/roomingInfo.html"))
)

func handleViewInvitation(wr WrappedRequest, invitationKey *datastore.Key) {
	var invitation Invitation
	err := datastore.Get(wr.Context, invitationKey, &invitation)
	if err != nil {
		log.Errorf(wr.Context, "error getting invitation: %v", err)
	}

	formInfoMap := make(map[*datastore.Key]PersonUpdateFormInfo)
	realizedInvitation := makeRealizedInvitation(wr.Context, invitationKey, &invitation)
	for i, invitee := range realizedInvitation.Invitees {
		personKey := invitee.Person.DatastoreKey
		formInfo := makePersonUpdateFormInfo(personKey, invitee.Person, i, true)
		formInfoMap[personKey] = formInfo
	}

	realActivities, err := activity.Realize(wr.Context, realizedInvitation.Event.Activities)
	if err != nil {
		log.Errorf(wr.Context, "activity.Realize: %v", err)
	}

	data := wr.MakeTemplateData(map[string]interface{}{
		"Invitation":                   realizedInvitation,
		"FormInfoMap":                  formInfoMap,
		"AllRsvpStatuses":              GetAllRsvpStatuses(),
		"Activities":                   realActivities,
		"AllHousingPreferences":        GetAllHousingPreferences(),
		"AllHousingPreferenceBooleans": GetAllHousingPreferenceBooleans(),
		"AllDrivingPreferences":        GetAllDrivingPreferences(),
		"AllParkingTypes":              GetAllParkingTypes(),
		"InvitationHasChildren":        invitation.HasChildren(wr.Context),
		"IsAdminUser":                  wr.IsAdminUser(),
		"RoomingInfo":                  getRoomingInfo(wr, invitationKey),
	})

	if err := invitationTpl.ExecuteTemplate(wr.ResponseWriter, "viewInvitation.html", data); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}

func HasPreference(total int, mask int) bool {
	return (total & mask) != 0
}

func handleSaveInvitation(wr WrappedRequest) {
	//ctx := appengine.NewContext(wr.Request)
	wr.Request.ParseForm()

	invitationKeyEncoded := wr.Request.Form.Get("invitation")
	invitationKey, _ := datastore.DecodeKey(invitationKeyEncoded)

	if !(wr.IsAdminUser() || *wr.InvitationKey == *invitationKey) {
		http.Error(wr.ResponseWriter,
			"Not authorized to edit invitation.",
			http.StatusForbidden)
		return
	}

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

	thursdayDinnerCount, err := strconv.Atoi(wr.Request.Form.Get("ThursdayDinnerCount"))
	if err == nil {
		invitation.ThursdayDinnerCount = thursdayDinnerCount
	} else {
		invitation.ThursdayDinnerCount = 0
	}
	fridayLunch := wr.Request.Form.Get("FridayLunch")
	invitation.FridayLunch = (fridayLunch == "on")

	fridayDinnerCount, err := strconv.Atoi(wr.Request.Form.Get("FridayDinnerCount"))
	if err == nil {
		invitation.FridayDinnerCount = fridayDinnerCount
	} else {
		invitation.FridayDinnerCount = 0
	}
	fridayIceCreamCount, err := strconv.Atoi(wr.Request.Form.Get("FridayIceCreamCount"))
	if err == nil {
		invitation.FridayIceCreamCount = fridayIceCreamCount
	} else {
		invitation.FridayIceCreamCount = 0
	}

	invitation.LastUpdatedPerson = wr.LoginInfo.PersonKey

	invitation.LastUpdatedTimestamp = time.Now()

	_, err = datastore.Put(wr.Context, invitationKey, &invitation)
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

	var isAttending []bool
	for _, invitee := range invitation.Invitees {
		if rsvp, present := rsvpMap[invitee]; present {
			attending := GetAllRsvpStatuses()[rsvp].Attending
			isAttending = append(isAttending, attending)
		} else {
			isAttending = append(isAttending, false)
		}
	}

	var e Event
	datastore.Get(wr.Context, invitation.Event, &e)
	subject := fmt.Sprintf("%s:%s RSVP from %s", e.ShortName, newPeopleSubjectFragment, CollectiveAddress(invitees, Informal))

	realizedInvitation := makeRealizedInvitation(wr.Context, invitationKey, &invitation)
	// TODO: escape this.
	//realizedInvitation.HousingNotes = strings.Replace(realizedInvitation.HousingNotes, "\n", "<br>", -1)

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
		AnyAttending:                 invitation.AnyAttending(),
		IsAttending:                  isAttending,
	}

	header := MailHeaderInfo{
		To:      []string{wr.GetSenderAddress()},
		Subject: subject,
	}

	sendMail(wr, "rsvpconfirmation", data, header)

	if !wr.IsAdminUser() {

		data := wr.MakeTemplateData(map[string]interface{}{
			"AnyAttending": invitation.AnyAttending(),
			"AnyUndecided": invitation.AnyUndecided(),
		})

		tpl := template.Must(template.ParseFiles("templates/main.html", "templates/thanks.html"))
		if err := tpl.ExecuteTemplate(wr.ResponseWriter, "thanks.html", data); err != nil {
			log.Errorf(wr.Context, "%v", err)
		}

		return
	}

	http.Redirect(wr.ResponseWriter, wr.Request, "invitations", http.StatusSeeOther)
}

func (invitation *Invitation) ClusterByRsvp(ctx context.Context) (map[RsvpStatus][]Person, []Person) {
	var personKeyToRsvp = make(map[datastore.Key]RsvpStatus)
	for p, r := range invitation.RsvpMap {
		personKeyToRsvp[*p] = r
	}

	rsvpMap := make(map[RsvpStatus][]Person)
	var noRsvp []Person

	for _, invitee := range invitation.Invitees {
		var person Person
		datastore.Get(ctx, invitee, &person)
		person.DatastoreKey = invitee

		if rsvp, present := personKeyToRsvp[*invitee]; present {
			listForStatus := rsvpMap[rsvp]
			if listForStatus == nil {
				listForStatus = make([]Person, 0)
			}
			listForStatus = append(listForStatus, person)
			rsvpMap[rsvp] = listForStatus

		} else {
			noRsvp = append(noRsvp, person)
		}
	}

	return rsvpMap, noRsvp
}
