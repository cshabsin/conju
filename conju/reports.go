package conju

import (
	"fmt"
	"html/template"
	"math"
	"net/http"
	"sort"

	"github.com/cshabsin/conju/activity"
	"github.com/cshabsin/conju/invitation"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

// func handleReports(wr WrappedRequest) {
// 	var tpl = template.Must(template.ParseFiles("templates/main.html", "templates/reports.html"))
// 	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "reports.html", wr.TemplateData); err != nil {
// 		log.Errorf(wr.Context, "%v", err)
// 	}
// }

func handleRsvpReport(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	currentEventKey := wr.EventKey

	var invitations []*Invitation
	q := datastore.NewQuery("Invitation").Filter("Event =", currentEventKey)
	invitationKeys, err := q.GetAll(ctx, &invitations)
	if err != nil {
		log.Errorf(ctx, "fetching invitations: %v", err)
	}

	allRsvpMap := make(map[invitation.RsvpStatus][][]Person)
	var allNoRsvp [][]Person

	type ExtraInvitationInfo struct {
		InvitationKey       string
		ThursdayDinnerCount int
		FridayLunch         bool
		FridayDinnerCount   int
		FridayIceCreamCount int
		TotalCost           float64
	}

	thursdayDinnerCount := 0
	yesFridayLunch := 0
	noFridayLunch := 0
	fridayDinnerCount := 0
	fridayIceCreamCount := 0

	personToExtraInfoMap := make(map[int64]*ExtraInvitationInfo)

	var bookings []Booking
	b := datastore.NewQuery("Booking").Ancestor(wr.EventKey)
	_, _ = b.GetAll(ctx, &bookings)

	personToCost := make(map[int64]float64)
	personToRsvpStatus := make(map[int64]invitation.RsvpStatus)
	personIdToPerson := make(map[int64]Person)

	for i, inv := range invitations {
		rsvpMap, noRsvp := inv.ClusterByRsvp(ctx)

		thursdayDinnerCount += inv.ThursdayDinnerCount
		fridayDinnerCount += inv.FridayDinnerCount
		fridayIceCreamCount += inv.FridayIceCreamCount
		thursdayCount := len(rsvpMap[invitation.ThuFriSat])
		if inv.FridayLunch {
			yesFridayLunch += thursdayCount
		} else {
			noFridayLunch += thursdayCount
		}

		ExtraInfo := ExtraInvitationInfo{
			InvitationKey:       invitationKeys[i].Encode(),
			ThursdayDinnerCount: inv.ThursdayDinnerCount,
			FridayLunch:         inv.FridayLunch,
			FridayDinnerCount:   inv.FridayDinnerCount,
			FridayIceCreamCount: inv.FridayIceCreamCount,
		}

		for r, p := range rsvpMap {
			for _, personForStatus := range p {
				personToRsvpStatus[personForStatus.DatastoreKey.IntID()] = r
				personIdToPerson[personForStatus.DatastoreKey.IntID()] = personForStatus
			}
			listOfLists := allRsvpMap[r]
			if listOfLists == nil {
				listOfLists = make([][]Person, 0)
			}
			listOfLists = append(listOfLists, p)
			allRsvpMap[r] = listOfLists
			personToExtraInfoMap[p[0].DatastoreKey.IntID()] = &ExtraInfo

		}
		if len(noRsvp) > 0 {
			allNoRsvp = append(allNoRsvp, noRsvp)
		}
	}

	for _, p := range allRsvpMap {
		sort.Slice(p, func(a, b int) bool { return SortByLastFirstName(p[a][0], p[b][0]) })
	}

	sort.Slice(allNoRsvp, func(a, b int) bool { return SortByLastFirstName(allNoRsvp[a][0], allNoRsvp[b][0]) })
	statusOrder := []invitation.RsvpStatus{invitation.ThuFriSat, invitation.FriSat, invitation.Maybe, invitation.No}

	for _, booking := range bookings {

		FridaySaturday := 0
		PlusThursday := 0
		addThurs := make([]bool, len(booking.Roommates))

		for i, person := range booking.Roommates {
			rsvpStatus := personToRsvpStatus[person.IntID()]
			if personIdToPerson[person.IntID()].IsBabyAtTime(wr.Event.StartDate) {
				continue
			}
			if rsvpStatus == invitation.FriSat {
				FridaySaturday++
			}
			if rsvpStatus == invitation.ThuFriSat {
				FridaySaturday++
				PlusThursday++
				addThurs[i] = true
			}
		}

		for i, person := range booking.Roommates {
			if personIdToPerson[person.IntID()].IsBabyAtTime(wr.Event.StartDate) {
				personToCost[person.IntID()] = 0
				continue
			}
			costForPerson := float64(0)
			if FridaySaturday <= 5 {
				costForPerson = invitation.GetAllRsvpStatuses()[invitation.FriSat].BaseCost[FridaySaturday]
			}

			if addThurs[i] && PlusThursday <= 5 {
				costForPerson += invitation.GetAllRsvpStatuses()[invitation.ThuFriSat].AddOnCost[PlusThursday]
			}
			costForPerson = math.Floor(costForPerson*100) / 100
			personToCost[person.IntID()] = costForPerson
		}
	}

	for status, personLists := range allRsvpMap {
		if !invitation.GetAllRsvpStatuses()[status].Attending {
			continue
		}
		for _, personList := range personLists {
			totalCost := float64(0)
			for _, person := range personList {
				totalCost += personToCost[person.DatastoreKey.IntID()]
			}
			personToExtraInfoMap[personList[0].DatastoreKey.IntID()].TotalCost = math.Floor(totalCost*100) / 100
		}

	}

	tpl := template.Must(template.New("").ParseFiles("templates/main.html", "templates/rsvpReport.html"))
	data := wr.MakeTemplateData(map[string]interface{}{
		"RsvpMap":              allRsvpMap,
		"NoRsvp":               allNoRsvp,
		"StatusOrder":          statusOrder,
		"AllRsvpStatuses":      invitation.GetAllRsvpStatuses(),
		"ThursdayDinnerCount":  thursdayDinnerCount,
		"FridayLunchYes":       yesFridayLunch,
		"FridayLunchNo":        noFridayLunch,
		"FridayDinnerCount":    fridayDinnerCount,
		"FridayIceCreamCount":  fridayIceCreamCount,
		"PersonToExtraInfoMap": personToExtraInfoMap,
		"PersonToCost":         personToCost,
	})
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "rsvpReport.html", data); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}

func handleActivitiesReport(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	currentEventKey := wr.EventKey

	var invitations []*Invitation
	q := datastore.NewQuery("Invitation").Filter("Event =", currentEventKey)
	_, err := q.GetAll(ctx, &invitations)
	if err != nil {
		log.Errorf(ctx, "fetching invitations: %v", err)
	}

	allRsvpStatuses := invitation.GetAllRsvpStatuses()

	activities, err := activity.Realize(ctx, wr.Event.Activities)
	if err != nil {
		log.Errorf(ctx, "fetching activities: %v", err)
	}

	keysToActivities := make(map[datastore.Key]*activity.Activity)
	activityKeys := make([]datastore.Key, len(wr.Event.Activities))
	for i, key := range wr.Event.Activities {
		keysToActivities[*key] = activities[i]
		activityKeys[i] = *key
	}

	type ActivityResponse struct {
		NoResponses         []datastore.Key
		MaybeResponses      []datastore.Key
		DefinitelyResponses []datastore.Key
		Leaders             []datastore.Key
		Expected            float64
	}

	activityResponseMap := make(map[datastore.Key]*ActivityResponse)
	for _, activityKey := range wr.Event.Activities {
		activityResponseMap[*activityKey] = &ActivityResponse{}
	}

	var allPeopleToLookUp []*datastore.Key
	for _, invitation := range invitations {
		if invitation.RsvpMap == nil {
			continue
		}

		if invitation.ActivityMap == nil {
			continue
		}

		personKeySet := make(map[datastore.Key]bool)
		for k, v := range invitation.RsvpMap {
			if allRsvpStatuses[v].Attending {
				personKeySet[*k] = true
				allPeopleToLookUp = append(allPeopleToLookUp, k)
			}
		}

		for k, v := range invitation.ActivityMap {

			if _, present := personKeySet[*k]; present {
				for ak, preference := range v {
					response := activityResponseMap[*ak]
					switch preference {
					case ActivityNo:
						response.NoResponses = append(response.NoResponses, *k)
					case ActivityMaybe:
						response.MaybeResponses = append(response.MaybeResponses, *k)
						response.Expected += .5
					case ActivityDefinitely:
						response.DefinitelyResponses = append(response.DefinitelyResponses, *k)
						response.Expected++
					}
				}
			}
		}

		for k, v := range invitation.ActivityLeaderMap {
			if _, present := personKeySet[*k]; present {
				for ak, leader := range v {
					response := activityResponseMap[*ak]
					if leader {
						response.Leaders = append(response.Leaders, *k)
					}
				}
			}
		}

	}

	var people = make([]*Person, len(allPeopleToLookUp))
	err = datastore.GetMulti(ctx, allPeopleToLookUp, people)
	if err != nil {
		log.Errorf(ctx, "fetching people: %v", err)
	}

	personMap := make(map[datastore.Key]string)
	for i, person := range people {
		personMap[*allPeopleToLookUp[i]] = person.FullNameWithAge(wr.Event.StartDate)
	}

	tpl := template.Must(template.New("").ParseFiles("templates/main.html", "templates/activitiesReport.html"))
	data := wr.MakeTemplateData(map[string]interface{}{
		"ActivityKeys":        activityKeys,
		"KeysToActivities":    keysToActivities,
		"ActivityResponseMap": activityResponseMap,
		"PersonMap":           personMap,
	})
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "activitiesReport.html", data); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}

}

func handleRoomingReport(wr WrappedRequest) {
	ctx := wr.Context

	var bookings []Booking
	q := datastore.NewQuery("Booking").Ancestor(wr.EventKey)
	bookingKeys, _ := q.GetAll(ctx, &bookings)

	roomsMap := make(map[int64]Room)
	var rooms = make([]*Room, len(wr.Event.Rooms))
	err := datastore.GetMulti(ctx, wr.Event.Rooms, rooms)
	if err != nil {
		log.Infof(ctx, "%v", err)
	}

	for i, room := range rooms {
		roomsMap[wr.Event.Rooms[i].IntID()] = *room
	}

	var buildingOrderMap = make(map[int64]int)
	for _, room := range wr.Event.Rooms {
		buildingKeyId := room.Parent().IntID()
		if _, present := buildingOrderMap[buildingKeyId]; !present {
			buildingOrderMap[buildingKeyId] = len(buildingOrderMap)
		}
	}

	var peopleToLookUp []*datastore.Key
	for _, booking := range bookings {
		peopleToLookUp = append(peopleToLookUp, booking.Roommates...)
	}

	personMap := make(map[int64]Person)
	var people = make([]*Person, len(peopleToLookUp))
	err = datastore.GetMulti(ctx, peopleToLookUp, people)
	if err != nil {
		log.Infof(ctx, "%v", err)
	}

	for i, person := range people {
		personMap[peopleToLookUp[i].IntID()] = *person
	}

	var invitations []*Invitation
	q = datastore.NewQuery("Invitation").Filter("Event =", wr.EventKey)
	invitationKeys, err := q.GetAll(ctx, &invitations)
	if err != nil {
		log.Errorf(ctx, "fetching invitations: %v", err)
	}

	personToInvitationMap := make(map[int64]int64)
	invitationMap := make(map[int64]Invitation)
	personToRsvpStatus := make(map[int64]invitation.RsvpStatus)
	for i, inv := range invitations {
		invitationMap[invitationKeys[i].IntID()] = *inv
		if inv.RsvpMap != nil {
			for p, r := range inv.RsvpMap {
				personToRsvpStatus[(*p).IntID()] = r
			}
		}
		for _, person := range inv.Invitees {
			personToInvitationMap[person.IntID()] = invitationKeys[i].IntID()
		}
	}

	shareBedBit := GetAllHousingPreferenceBooleans()[ShareBed].Bit

	type RealBooking struct {
		KeyString           string
		Room                Room
		Building            *Building
		Roommates           []Person
		ShowConvertToDouble bool
		FriSat              int
		PlusThurs           int
		AddThurs            []bool
		Cost                float64
		CostString          string
		Reserved            bool
	}

	buildingsMap := getBuildingMapForVenue(wr.Context, wr.Event.Venue)
	// doesn't deal with consolidating partitioned rooms
	var realBookingsByBuilding = make([][]RealBooking, len(buildingOrderMap))
	var totalCostForEveryone float64
	for i, booking := range bookings {
		people := make([]Person, len(booking.Roommates))

		FridaySaturday := 0
		PlusThursday := 0
		addThurs := make([]bool, len(booking.Roommates))
		doubleBedNeeded := false // for now don't deal with more than one double needed

		for i, person := range booking.Roommates {
			people[i] = personMap[person.IntID()]
			inv := invitationMap[personToInvitationMap[person.IntID()]]
			doubleBedNeeded = doubleBedNeeded || (inv.HousingPreferenceBooleans&shareBedBit == shareBedBit)
			rsvpStatus := personToRsvpStatus[person.IntID()]
			if people[i].IsBabyAtTime(wr.Event.StartDate) {
				continue
			}
			if rsvpStatus == invitation.FriSat {
				FridaySaturday++
			}
			if rsvpStatus == invitation.ThuFriSat {
				FridaySaturday++
				PlusThursday++
				addThurs[i] = true
			}

		}

		room := roomsMap[booking.Room.IntID()]
		buildingId := booking.Room.Parent().IntID()
		building := buildingsMap[buildingId]
		// for now? ignore case where they want a double bed and aren't getting it
		showConvertToDouble := doubleBedNeeded

		if doubleBedNeeded && (((building.Properties | room.Properties) & shareBedBit) == shareBedBit) {
			for _, bed := range room.Beds {
				if bed == Double || bed == Queen || bed == King {
					showConvertToDouble = false
					break
				}
			}
		}

		basePPCost := float64(0)
		basePPCostString := "???"
		if FridaySaturday <= 5 {
			basePPCost = invitation.GetAllRsvpStatuses()[invitation.FriSat].BaseCost[FridaySaturday]
			basePPCostString = fmt.Sprintf("$%.2f", basePPCost)
		}
		addOnPPCost := float64(0)
		addOnPPCostString := "???"
		if PlusThursday <= 5 {
			addOnPPCost = invitation.GetAllRsvpStatuses()[invitation.ThuFriSat].AddOnCost[PlusThursday]
			addOnPPCostString = fmt.Sprintf("$%.2f", addOnPPCost)
		}

		addOnTotalString := ""
		if PlusThursday > 0 {
			addOnTotalString = fmt.Sprintf(" + %d * %s", PlusThursday, addOnPPCostString)
		}

		totalCost := float64(FridaySaturday)*basePPCost + float64(PlusThursday)*addOnPPCost
		totalCostForEveryone += totalCost
		totalCostString := fmt.Sprintf("$%.2f", totalCost)
		if FridaySaturday > 5 || PlusThursday > 5 {
			totalCostString = "???"
		}

		costEquationString := fmt.Sprintf("%d * %s %s = %s", FridaySaturday, basePPCostString, addOnTotalString, totalCostString)

		realBooking := RealBooking{
			KeyString:           bookingKeys[i].Encode(),
			Room:                roomsMap[booking.Room.IntID()],
			Building:            building,
			Roommates:           people,
			FriSat:              FridaySaturday,
			PlusThurs:           PlusThursday,
			AddThurs:            addThurs,
			ShowConvertToDouble: showConvertToDouble,
			Cost:                totalCost,
			CostString:          costEquationString,
			Reserved:            booking.Reserved,
		}
		buildingIndex := buildingOrderMap[buildingId]
		bookingsForBuilding := realBookingsByBuilding[buildingIndex]
		if bookingsForBuilding == nil {
			bookingsForBuilding = make([]RealBooking, 0)
		}
		bookingsForBuilding = append(bookingsForBuilding, realBooking)
		realBookingsByBuilding[buildingIndex] = bookingsForBuilding
	}

	for _, buildingGroup := range realBookingsByBuilding {
		sort.Slice(buildingGroup,
			func(a, b int) bool {
				return buildingGroup[a].Room.RoomNumber < buildingGroup[b].Room.RoomNumber
			})
	}

	tpl := template.Must(template.New("").ParseFiles("templates/main.html", "templates/roomingReport.html"))
	data := wr.MakeTemplateData(map[string]interface{}{
		"BookingsByBuilding":   realBookingsByBuilding,
		"TotalCostForEveryone": totalCostForEveryone,
	})
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "roomingReport.html", data); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}

func handleSaveReservations(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	wr.Request.ParseForm()

	bookingToBooked := make(map[int64]bool)

	for k, v := range wr.Request.PostForm {
		log.Infof(ctx, "key: %v", k)
		if v[0] == "" {
			continue
		}
		if k[0:8] == "booking_" {
			bookingKey, _ := datastore.DecodeKey(string(k[8:]))
			bookingToBooked[bookingKey.IntID()] = true
		}
	}

	var bookings []*Booking
	q := datastore.NewQuery("Booking").Ancestor(wr.EventKey)
	bookingKeys, _ := q.GetAll(ctx, &bookings)

	for i, booking := range bookings {
		if _, present := bookingToBooked[bookingKeys[i].IntID()]; present {
			booking.Reserved = true
		} else {
			booking.Reserved = false
		}
	}

	datastore.PutMulti(ctx, bookingKeys, bookings)

	http.Redirect(wr.ResponseWriter, wr.Request, "admin", http.StatusSeeOther)
}

func handleFoodReport(wr WrappedRequest) {
	ctx := wr.Context
	currentEventKey := wr.EventKey

	allRsvpStatuses := invitation.GetAllRsvpStatuses()
	totalRestrictions := len(GetAllFoodRestrictionTags())

	counts := make([]int, totalRestrictions)
	personToRestrictions := make(map[int64][]bool)
	var people []Person

	var invitations []*Invitation
	q := datastore.NewQuery("Invitation").Filter("Event =", currentEventKey)
	_, err := q.GetAll(ctx, &invitations)
	if err != nil {
		log.Errorf(ctx, "fetching invitations: %v", err)
	}

	for _, inv := range invitations {

		for p, s := range inv.RsvpMap {

			status := allRsvpStatuses[s]
			if status.Attending {
				var person Person
				datastore.Get(ctx, p, &person)
				people = append(people, person)
				restrictionsForPerson := make([]bool, totalRestrictions)
				for _, restriction := range person.FoodRestrictions {
					counts[restriction]++
					restrictionsForPerson[restriction] = true
				}
				personToRestrictions[p.IntID()] = restrictionsForPerson
			}
		}

	}

	sort.Slice(people, func(a, b int) bool { return SortByLastFirstName(people[a], people[b]) })

	tpl := template.Must(template.New("").ParseFiles("templates/main.html", "templates/foodReport.html"))
	data := wr.MakeTemplateData(map[string]interface{}{
		"AllRestrictions":      GetAllFoodRestrictionTags(),
		"Counts":               counts,
		"People":               people,
		"PersonToRestrictions": personToRestrictions,
	})

	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "foodReport.html", data); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}

func handleRidesReport(wr WrappedRequest) {
	ctx := wr.Context
	currentEventKey := wr.EventKey

	allRsvpStatuses := invitation.GetAllRsvpStatuses()

	type CarRequest struct {
		People               []*Person
		Preference           DrivingPreference
		AlsoDrive            bool
		LeaveFrom            string
		LeaveTime            string
		AdditionalPassengers string
		TravelNotes          string
	}

	var ThursdayDrivers []CarRequest
	var ThursdayRiders []CarRequest
	var FridayDrivers []CarRequest
	var FridayRiders []CarRequest

	var ThursdayIndependent []CarRequest
	var FridayIndependent []CarRequest

	var invitations []*Invitation
	q := datastore.NewQuery("Invitation").Filter("Event =", currentEventKey)
	_, err := q.GetAll(ctx, &invitations)
	if err != nil {
		log.Errorf(ctx, "fetching invitations: %v", err)
	}

	for _, inv := range invitations {

		var personKeys []*datastore.Key
		thursday := false
		for p, s := range inv.RsvpMap {
			status := allRsvpStatuses[s]
			if status.Attending {
				personKeys = append(personKeys, p)
			}
			// TODO: split rides by person
			if status.Status == invitation.ThuFriSat {
				thursday = true
			}
		}

		if len(personKeys) == 0 {
			continue
		}

		var people = make([]*Person, len(personKeys))
		err = datastore.GetMulti(ctx, personKeys, people)
		if err != nil {
			log.Errorf(ctx, "fetching people: %v", err)
		}

		request := CarRequest{
			People:               people,
			Preference:           (*inv).Driving,
			AlsoDrive:            (*inv).Driving == DriveIfNeeded,
			LeaveFrom:            (*inv).LeaveFrom,
			LeaveTime:            (*inv).LeaveTime,
			AdditionalPassengers: (*inv).AdditionalPassengers,
			TravelNotes:          (*inv).TravelNotes,
		}

		if (*inv).Driving == Driving {
			if thursday {
				ThursdayDrivers = append(ThursdayDrivers, request)
			} else {
				FridayDrivers = append(FridayDrivers, request)
			}

		} else if (*inv).Driving == DrivingNotSet || (*inv).Driving == NoCarpool {
			if thursday {
				ThursdayIndependent = append(ThursdayIndependent, request)
			} else {
				FridayIndependent = append(FridayIndependent, request)
			}
		} else {
			if thursday {
				ThursdayRiders = append(ThursdayRiders, request)
			} else {
				FridayRiders = append(FridayRiders, request)
			}
		}

	}

	tpl := template.Must(template.New("").ParseFiles("templates/main.html", "templates/ridesReport.html"))
	data := wr.MakeTemplateData(map[string]interface{}{
		"ThursdayDrivers":     ThursdayDrivers,
		"ThursdayRiders":      ThursdayRiders,
		"ThursdayIndependent": ThursdayIndependent,
		"FridayDrivers":       FridayDrivers,
		"FridayRiders":        FridayRiders,
		"FridayIndependent":   FridayIndependent,
	})

	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "ridesReport.html", data); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}
