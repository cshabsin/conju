package conju

// TODO: move to "package models"?

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/datastore"

	"github.com/cshabsin/conju/activity"
	"github.com/cshabsin/conju/conju/dsclient"
	"github.com/cshabsin/conju/invitation"
	"github.com/cshabsin/conju/model/event"
	"github.com/cshabsin/conju/model/person"
)

type Invitation struct {
	Event                     *datastore.Key                           // Event
	Invitees                  []*datastore.Key                         // []Person
	RsvpMap                   map[*datastore.Key]invitation.RsvpStatus // Person -> Rsvp
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
	COVIDAcked                bool
	Storyland                 bool
}

const delimiter = "|_|"

func (inv *Invitation) Load(ps []datastore.Property) error {
	allRsvpStatuses := invitation.GetAllRsvpStatuses()

	inv.RsvpMap = make(map[*datastore.Key]invitation.RsvpStatus)
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
				log.Printf("person lookup error: %v", err)
				continue
			}
			if personKey == nil {
				log.Printf("person lookup yielded nil key for map entry %s", p.Name)
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
				log.Printf("person lookup error: %v", err)
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
		{
			Name:  "COVIDAcked",
			Value: inv.COVIDAcked,
		},
		{
			Name:  "Storyland",
			Value: inv.Storyland,
		},
	}

	for _, invitee := range inv.Invitees {
		inviteeProp := datastore.Property{
			Name:  "Invitees",
			Value: invitee,
			// Multiple: true, // TODO: is this safe?
		}
		props = append(props, inviteeProp)
	}

	rsvpMap := inv.RsvpMap
	for k, v := range rsvpMap {
		encodedKey := k.Encode()
		props = append(props, datastore.Property{Name: "RsvpMap." + encodedKey, Value: int64(v)})
	}

	activityMap := inv.ActivityMap
	for p, m := range activityMap {
		personEncodedKey := p.Encode()
		partialName := "ActivityMap." + personEncodedKey
		for a, v := range m {
			totalKey := partialName + delimiter + (*a).Encode()
			props = append(props, datastore.Property{Name: totalKey, Value: int64(v)})
		}
	}
	activityLeaderMap := inv.ActivityLeaderMap
	for p, m := range activityLeaderMap {
		personEncodedKey := p.Encode()
		partialName := "ActivityLeaderMap." + personEncodedKey
		for a, v := range m {
			totalKey := partialName + delimiter + (*a).Encode()
			props = append(props, datastore.Property{Name: totalKey, Value: v})
		}
	}

	return props, nil
}

func (inv *Invitation) AnyAttending() bool {
	allStatuses := invitation.GetAllRsvpStatuses()
	for _, v := range inv.RsvpMap {
		attending := allStatuses[v].Attending
		if attending {
			return attending
		}
	}
	return false
}

func (inv *Invitation) AttendingInvitees() []*datastore.Key {
	allStatuses := invitation.GetAllRsvpStatuses()
	var attending []*datastore.Key
	for k, v := range inv.RsvpMap {
		if allStatuses[v].Attending {
			attending = append(attending, k)
		}
	}
	return attending
}

func (inv *Invitation) AnyUndecided() bool {
	allStatuses := invitation.GetAllRsvpStatuses()
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
	event, err := event.GetEvent(ctx, inv.Event)
	if err != nil {
		log.Printf(
			"GetEvent: %v", err)
	}

	for _, personKey := range inv.Invitees {
		var person person.Person
		dsclient.FromContext(ctx).Get(ctx, personKey, &person)
		if person.IsNonAdultAtTime(event.StartDate) {
			return true
		}
	}
	return false
}

// Handles /invitations, listing invitations.
func handleInvitations(ctx context.Context, wr WrappedRequest) {
	currentEventKey := wr.EventKey

	var notInvitedSet = make(map[datastore.Key]person.PersonWithKey)
	personQuery := datastore.NewQuery("Person")
	var people []*person.Person
	personKeys, _ := dsclient.FromContext(ctx).GetAll(ctx, personQuery, &people)

	for i := 0; i < len(personKeys); i++ {
		personWithKey := person.PersonWithKey{Key: personKeys[i].Encode(), Person: *people[i]}
		notInvitedSet[*personKeys[i]] = personWithKey
	}

	var invitations []*Invitation

	q := datastore.NewQuery("Invitation").FilterField("Event", "=", currentEventKey)
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		log.Printf(
			"fetching invitations: %v", err)
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
				age := person.HalfYears(invitee.Person.ApproxAgeAtTime(wr.Event.StartDate))
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
			return person.SortByLastFirstName(realizedInvitations[a].Invitees[0].Person, realizedInvitations[b].Invitees[0].Person)
		})

	notInvitedList := make([]person.PersonWithKey, len(notInvitedSet))
	i := 0
	for k := range notInvitedSet {
		notInvitedList[i] = notInvitedSet[k]
		i++
	}
	sort.Slice(notInvitedList, func(a, b int) bool {
		return person.SortByLastFirstName(notInvitedList[a].Person, notInvitedList[b].Person)
	})

	var allEvents []*event.Event
	if len(invitations) == 0 {
		var err error
		allEvents, err = event.GetAllEvents(ctx)
		if err != nil {
			log.Printf(
				"GetAllEvents: %v", err)
		}
	}

	data := wr.MakeTemplateData(map[string]interface{}{
		"Invitations":         invitations,
		"RealizedInvitations": realizedInvitations,
		"NotInvitedList":      notInvitedList,
		"AllEvents":           allEvents,
		"Stats":               statistics,
	})

	functionMap := template.FuncMap{
		"ListInvitees": func(peopleWithKeys []person.PersonWithKey) string {
			var people []person.Person
			for _, person := range peopleWithKeys {
				people = append(people, person.Person)
			}
			return person.CollectiveAddress(people, person.Informal)
		},
	}

	tpl := template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/main.html", "templates/invitations.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "invitations.html", data); err != nil {
		log.Printf("%v", err)
	}
}

func handleCopyInvitations(ctx context.Context, wr WrappedRequest) {
	currentEventKey := wr.EventKey
	wr.Request.ParseForm()

	baseEventKeyEncoded := wr.Request.Form.Get("baseEvent")
	log.Printf("Found base event: %v", baseEventKeyEncoded)
	if baseEventKeyEncoded == "" {
		return
	}

	baseEventKey, err := datastore.DecodeKey(baseEventKeyEncoded)
	if err != nil {
		log.Printf("error decoding event key: %v", err)
	}
	var invitations []*Invitation
	q := datastore.NewQuery("Invitation").FilterField("Event", "=", baseEventKey)
	dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)

	log.Printf("Found %d invitations from copied event", len(invitations))
	var newInvitations []Invitation
	var newInvitationKeys []*datastore.Key
	for _, invitation := range invitations {
		newInvitations = append(newInvitations, Invitation{
			Event:    currentEventKey,
			Invitees: invitation.Invitees,
		})
		newKey := datastore.IncompleteKey("Invitation", nil)
		newInvitationKeys = append(newInvitationKeys, newKey)
	}

	_, error := dsclient.FromContext(ctx).PutMulti(ctx, newInvitationKeys, newInvitations)
	if error != nil {
		log.Printf("Error in putmulti: %v", error)
	}
	http.Redirect(wr.ResponseWriter, wr.Request, "invitations", http.StatusSeeOther)

}

func handleAddInvitation(ctx context.Context, wr WrappedRequest) {
	currentEventKey := wr.EventKey
	wr.Request.ParseForm()

	invitationKeyEncoded := wr.Request.Form.Get("invitation")
	people := wr.Request.Form["person"]

	if len(people) == 0 {
		log.Printf("Couldn't find any selected people!")
		return
	}

	var newPeople []*datastore.Key
	for _, person := range people {
		key, _ := datastore.DecodeKey(person)
		newPeople = append(newPeople, key)
	}

	if invitationKeyEncoded == "" {
		log.Printf("no invitation selected, creating new one...")
		newKey := datastore.IncompleteKey("Invitation", nil)
		var newInvitation Invitation
		newInvitation.Event = currentEventKey
		newInvitation.Invitees = newPeople

		_, err := dsclient.FromContext(ctx).Put(ctx, newKey, &newInvitation)
		if err != nil {
			log.Printf("%v", err)
		}
	} else {
		existingInvitationKey, _ := datastore.DecodeKey(invitationKeyEncoded)
		var existingInvitation Invitation
		dsclient.FromContext(ctx).Get(ctx, existingInvitationKey, &existingInvitation)
		existingInvitation.Invitees = append(existingInvitation.Invitees, newPeople...)
		_, err := dsclient.FromContext(ctx).Put(ctx, existingInvitationKey, &existingInvitation)
		if err != nil {
			log.Printf("%v", err)
		}
	}

	http.Redirect(wr.ResponseWriter, wr.Request, "invitations", http.StatusSeeOther)
}

func handleDeleteInvitation(ctx context.Context, wr WrappedRequest) {
	wr.Request.ParseForm()

	invitationKeyEncoded := wr.Request.Form.Get("invitation")
	invitationKey, err := datastore.DecodeKey(invitationKeyEncoded)
	if err != nil {
		log.Printf("key decryption error: %v", err)
	}

	err = dsclient.FromContext(ctx).Delete(ctx, invitationKey)
	if err != nil {
		log.Printf("invitation deletion error: %v", err)
	}
	http.Redirect(wr.ResponseWriter, wr.Request, "invitations", http.StatusSeeOther)
}

// handleViewInvitationUser handles /viewInvitation URLs.
func handleViewInvitationAdmin(ctx context.Context, wr WrappedRequest) {
	wr.Request.ParseForm()

	invitationKeyEncoded := wr.Request.Form.Get("invitation")
	invitationKey, err := datastore.DecodeKey(invitationKeyEncoded)
	if err != nil {
		http.Error(wr.ResponseWriter,
			fmt.Sprintf("Error decoding invitation key: %v", err),
			http.StatusBadRequest)
	}
	handleViewInvitation(ctx, wr, invitationKey)
}

// handleViewInvitationUser handles /rsvp URLs.
func handleViewInvitationUser(ctx context.Context, wr WrappedRequest) {
	log.Printf("in handleViewInvitationUser")
	handleViewInvitation(ctx, wr, wr.InvitationKey)
}

var (
	functionMap = template.FuncMap{
		"PronounString":               person.GetPronouns,
		"HasPreference":               HasPreference,
		"DerefPeople":                 DerefPeople,
		"CollectiveAddressFirstNames": person.CollectiveAddressFirstNames,
		"SharerName":                  MakeSharerName,
	}

	invitationTpl = template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/main.html", "templates/viewInvitation.html", "templates/updatePersonForm.html", "templates/roomingInfo.html"))
)

func handleViewInvitation(ctx context.Context, wr WrappedRequest, invitationKey *datastore.Key) {
	var inv Invitation
	err := dsclient.FromContext(ctx).Get(ctx, invitationKey, &inv)
	if err != nil {
		log.Printf("error getting invitation: %v", err)
	}

	formInfoMap := make(map[*datastore.Key]person.PersonUpdateFormInfo)
	realizedInvitation := makeRealizedInvitation(ctx, invitationKey, &inv)
	for i, invitee := range realizedInvitation.Invitees {
		personKey := invitee.Person.DatastoreKey
		formInfo := person.MakePersonUpdateFormInfo(personKey, invitee.Person, i, true)
		formInfoMap[personKey] = formInfo
	}

	realActivities, err := activity.Realize(ctx, realizedInvitation.Event.Activities)
	if err != nil {
		log.Printf("activity.Realize: %v", err)
	}

	data := wr.MakeTemplateData(map[string]interface{}{
		"Invitation":                   realizedInvitation,
		"FormInfoMap":                  formInfoMap,
		"AllRsvpStatuses":              invitation.GetAllRsvpStatuses(),
		"Activities":                   realActivities,
		"AllHousingPreferences":        GetAllHousingPreferences(),
		"AllHousingPreferenceBooleans": GetAllHousingPreferenceBooleans(),
		"AllDrivingPreferences":        GetAllDrivingPreferences(),
		"AllParkingTypes":              GetAllParkingTypes(),
		"InvitationHasChildren":        inv.HasChildren(ctx),
		"IsAdminUser":                  wr.IsAdminUser(),
		"RoomingInfo":                  getRoomingInfo(ctx, wr, invitationKey),
	})

	if err := invitationTpl.ExecuteTemplate(wr.ResponseWriter, "viewInvitation.html", data); err != nil {
		log.Printf("%v", err)
	}
}

func HasPreference(total int, mask int) bool {
	return (total & mask) != 0
}

func handleSaveInvitation(ctx context.Context, wr WrappedRequest) {
	wr.Request.ParseForm()

	invitationKeyEncoded := wr.Request.Form.Get("invitation")
	invitationKey, _ := datastore.DecodeKey(invitationKeyEncoded)

	if !(wr.IsAdminUser() || *wr.InvitationKey == *invitationKey) {
		http.Error(wr.ResponseWriter,
			"Not authorized to edit invitation.",
			http.StatusForbidden)
		return
	}

	var inv Invitation
	dsclient.FromContext(ctx).Get(ctx, invitationKey, &inv)

	people := wr.Request.Form["person"]
	rsvps := wr.Request.Form["rsvp"]
	var newPeople []*datastore.Key
	var rsvpMap = make(map[*datastore.Key]invitation.RsvpStatus)
	var activityMap = make(map[*datastore.Key](map[*datastore.Key]ActivityRanking))
	var activityLeaderMap = make(map[*datastore.Key](map[*datastore.Key]bool))
	for i, personKey := range people {
		key, _ := datastore.DecodeKey(personKey)
		var person person.Person
		dsclient.FromContext(ctx).Get(ctx, key, &person)
		newPeople = append(newPeople, key)
		rsvp, _ := strconv.Atoi(rsvps[i])
		if rsvp >= 0 {
			fullStatus := invitation.GetAllRsvpStatuses()[rsvp]
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

	inv.RsvpMap = rsvpMap
	inv.ActivityMap = activityMap
	inv.ActivityLeaderMap = activityLeaderMap

	inv.Invitees = newPeople

	housingPreference, _ := strconv.Atoi(wr.Request.Form.Get("housingPreference"))
	if housingPreference >= 0 {
		hp := HousingPreference(housingPreference)
		inv.Housing = hp
	}

	var booleanInfos = GetAllHousingPreferenceBooleans()
	var housingPreferenceTotal int
	booleans := wr.Request.Form["housingPreferenceBooleans"]

	for _, boolean := range booleans {
		value, _ := strconv.Atoi(boolean)
		booleanInfo := booleanInfos[value]
		housingPreferenceTotal += booleanInfo.Bit
	}

	inv.HousingPreferenceBooleans = housingPreferenceTotal

	inv.HousingNotes = wr.Request.Form.Get("housingNotes")

	drivingPreference, _ := strconv.Atoi(wr.Request.Form.Get("drivingPreference"))
	inv.Driving = DrivingPreference(drivingPreference)

	storylandPreference := wr.Request.Form.Get("storylandPreference")
	inv.Storyland = storylandPreference == "yes"

	parkingType, _ := strconv.Atoi(wr.Request.Form.Get("parking"))
	if parkingType >= 0 {
		pt := ParkingType(parkingType)
		inv.Parking = pt
	}

	inv.LeaveFrom = wr.Request.Form.Get("leaveFrom")
	inv.LeaveTime = wr.Request.Form.Get("leaveTime")
	inv.AdditionalPassengers = wr.Request.Form.Get("additionalPassengers")
	inv.TravelNotes = wr.Request.Form.Get("travelNotes")
	inv.OtherInfo = wr.Request.Form.Get("otherInfo")
	log.Printf("covidAcked = %q", wr.Request.Form.Get("covidAcked"))
	inv.COVIDAcked = wr.Request.Form.Get("covidAcked") == "on"

	thursdayDinnerCount, err := strconv.Atoi(wr.Request.Form.Get("ThursdayDinnerCount"))
	if err == nil {
		inv.ThursdayDinnerCount = thursdayDinnerCount
	} else {
		inv.ThursdayDinnerCount = 0
	}
	fridayLunch := wr.Request.Form.Get("FridayLunch")
	inv.FridayLunch = (fridayLunch == "on")

	fridayDinnerCount, err := strconv.Atoi(wr.Request.Form.Get("FridayDinnerCount"))
	if err == nil {
		inv.FridayDinnerCount = fridayDinnerCount
	} else {
		inv.FridayDinnerCount = 0
	}
	fridayIceCreamCount, err := strconv.Atoi(wr.Request.Form.Get("FridayIceCreamCount"))
	if err == nil {
		inv.FridayIceCreamCount = fridayIceCreamCount
	} else {
		inv.FridayIceCreamCount = 0
	}

	inv.LastUpdatedPerson = wr.LoginInfo.PersonKey

	inv.LastUpdatedTimestamp = time.Now()

	_, err = dsclient.FromContext(ctx).Put(ctx, invitationKey, &inv)
	if err != nil {
		log.Printf("%v", err)
	}

	var invitees []person.Person
	for _, personKey := range inv.Invitees {
		var person person.Person
		dsclient.FromContext(ctx).Get(ctx, personKey, &person)
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
	for _, invitee := range inv.Invitees {
		if rsvp, present := rsvpMap[invitee]; present {
			attending := invitation.GetAllRsvpStatuses()[rsvp].Attending
			isAttending = append(isAttending, attending)
		} else {
			isAttending = append(isAttending, false)
		}
	}

	ev, err := event.GetEvent(ctx, inv.Event)
	if err != nil {
		log.Printf("GetEvent: %v", err)
	}
	subject := fmt.Sprintf("%s:%s RSVP from %s", ev.ShortName, newPeopleSubjectFragment, person.CollectiveAddress(invitees, person.Informal))

	realizedInvitation := makeRealizedInvitation(ctx, invitationKey, &inv)
	// TODO: escape this.
	//realizedInvitation.HousingNotes = strings.Replace(realizedInvitation.HousingNotes, "\n", "<br>", -1)

	data := struct {
		RealInvitation               RealizedInvitation
		AllHousingPreferenceBooleans []HousingPreferenceBooleanInfo
		AllPronouns                  []person.PronounSet
		AllFoodRestrictions          []person.FoodRestrictionTag
		AdditionalPeople             []NewPersonInfo
		AnyAttending                 bool
		IsAttending                  []bool
	}{
		RealInvitation:               realizedInvitation,
		AllHousingPreferenceBooleans: GetAllHousingPreferenceBooleans(),
		AllPronouns:                  []person.PronounSet{person.They, person.She, person.He, person.Zie},
		AllFoodRestrictions:          person.GetAllFoodRestrictionTags(),
		AdditionalPeople:             additionalPeople,
		AnyAttending:                 inv.AnyAttending(),
		IsAttending:                  isAttending,
	}

	header := MailHeaderInfo{
		To:      []string{wr.GetSenderAddress()},
		Subject: subject,
		BccSelf: false,
	}

	sendMail(wr, "rsvpconfirmation", data, header)

	if !wr.IsAdminUser() {

		data := wr.MakeTemplateData(map[string]interface{}{
			"AnyAttending": inv.AnyAttending(),
			"AnyUndecided": inv.AnyUndecided(),
		})

		tpl := template.Must(template.ParseFiles("templates/main.html", "templates/thanks.html"))
		if err := tpl.ExecuteTemplate(wr.ResponseWriter, "thanks.html", data); err != nil {
			log.Printf("%v", err)
		}

		return
	}

	http.Redirect(wr.ResponseWriter, wr.Request, "invitations", http.StatusSeeOther)
}

func (inv *Invitation) ClusterByRsvp(ctx context.Context) (map[invitation.RsvpStatus][]person.Person, []person.Person) {
	var personKeyToRsvp = make(map[datastore.Key]invitation.RsvpStatus)
	for p, r := range inv.RsvpMap {
		personKeyToRsvp[*p] = r
	}

	rsvpMap := make(map[invitation.RsvpStatus][]person.Person)
	var noRsvp []person.Person

	for _, invitee := range inv.Invitees {
		var per person.Person
		dsclient.FromContext(ctx).Get(ctx, invitee, &per)
		per.DatastoreKey = invitee

		if rsvp, present := personKeyToRsvp[*invitee]; present {
			listForStatus := rsvpMap[rsvp]
			if listForStatus == nil {
				listForStatus = make([]person.Person, 0)
			}
			listForStatus = append(listForStatus, per)
			rsvpMap[rsvp] = listForStatus

		} else {
			noRsvp = append(noRsvp, per)
		}
	}

	return rsvpMap, noRsvp
}
