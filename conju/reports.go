package conju

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"sort"

	"cloud.google.com/go/datastore"

	"github.com/cshabsin/conju/activity"
	"github.com/cshabsin/conju/conju/dsclient"
	"github.com/cshabsin/conju/invitation"
	"github.com/cshabsin/conju/model/housing"
	"github.com/cshabsin/conju/model/person"
)

// func handleReports(wr WrappedRequest) {
// 	var tpl = template.Must(template.ParseFiles("templates/main.html", "templates/reports.html"))
// 	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "reports.html", wr.TemplateData); err != nil {
// 		log.Printf( "%v", err)
// 	}
// }

func handleRsvpReport(ctx context.Context, wr WrappedRequest) {
	currentEventKey := wr.EventKey

	var invitations []*Invitation
	q := datastore.NewQuery("Invitation").FilterField("Event", "=", currentEventKey)
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		log.Printf("fetching invitations: %v", err)
	}

	allRsvpMap := make(map[invitation.RsvpStatus][][]person.Person)
	var allNoRsvp [][]person.Person

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
	q = datastore.NewQuery("Booking").Ancestor(wr.EventKey)
	_, _ = dsclient.FromContext(ctx).GetAll(ctx, q, &bookings)

	personToCost := make(map[int64]float64)
	personToRsvpStatus := make(map[int64]invitation.RsvpStatus)
	personIdToPerson := make(map[int64]person.Person)

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
				personToRsvpStatus[personForStatus.DatastoreKey.ID] = r
				personIdToPerson[personForStatus.DatastoreKey.ID] = personForStatus
			}
			listOfLists := allRsvpMap[r]
			if listOfLists == nil {
				listOfLists = make([][]person.Person, 0)
			}
			listOfLists = append(listOfLists, p)
			allRsvpMap[r] = listOfLists
			personToExtraInfoMap[p[0].DatastoreKey.ID] = &ExtraInfo

		}
		if len(noRsvp) > 0 {
			allNoRsvp = append(allNoRsvp, noRsvp)
		}
	}

	for _, p := range allRsvpMap {
		sort.Slice(p, func(a, b int) bool { return person.SortByLastFirstName(p[a][0], p[b][0]) })
	}

	sort.Slice(allNoRsvp, func(a, b int) bool { return person.SortByLastFirstName(allNoRsvp[a][0], allNoRsvp[b][0]) })
	statusOrder := []invitation.RsvpStatus{invitation.FriSatSun, invitation.FriSat, invitation.SatSun, invitation.Maybe, invitation.No}

	for _, booking := range bookings {

		FridaySaturday := 0
		PlusThursday := 0
		addThurs := make([]bool, len(booking.Roommates))

		for i, person := range booking.Roommates {
			rsvpStatus := personToRsvpStatus[person.ID]
			if personIdToPerson[person.ID].IsBabyAtTime(wr.Event.StartDate) {
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
			if personIdToPerson[person.ID].IsBabyAtTime(wr.Event.StartDate) {
				personToCost[person.ID] = 0
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
			personToCost[person.ID] = costForPerson
		}
	}

	for status, personLists := range allRsvpMap {
		if !invitation.GetAllRsvpStatuses()[status].Attending {
			continue
		}
		for _, personList := range personLists {
			totalCost := float64(0)
			for _, person := range personList {
				totalCost += personToCost[person.DatastoreKey.ID]
			}
			personToExtraInfoMap[personList[0].DatastoreKey.ID].TotalCost = math.Floor(totalCost*100) / 100
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
		log.Printf("%v", err)
	}
}

func handleActivitiesReport(ctx context.Context, wr WrappedRequest) {
	currentEventKey := wr.EventKey

	var invitations []*Invitation
	q := datastore.NewQuery("Invitation").FilterField("Event", "=", currentEventKey)
	_, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		log.Printf("fetching invitations: %v", err)
	}

	allRsvpStatuses := invitation.GetAllRsvpStatuses()

	activities, err := activity.Realize(ctx, wr.Event.Activities)
	if err != nil {
		log.Printf("fetching activities: %v", err)
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

	var people = make([]*person.Person, len(allPeopleToLookUp))
	err = dsclient.FromContext(ctx).GetMulti(ctx, allPeopleToLookUp, people)
	if err != nil {
		log.Printf("fetching people: %v", err)
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
		log.Printf("%v", err)
	}

}

func handleRoomingReport(ctx context.Context, wr WrappedRequest) {
	var bookings []Booking
	q := datastore.NewQuery("Booking").Ancestor(wr.EventKey)
	bookingKeys, _ := dsclient.FromContext(ctx).GetAll(ctx, q, &bookings)

	roomsMap := make(map[int64]housing.Room)
	var rooms = make([]*housing.Room, len(wr.Event.Rooms))
	err := dsclient.FromContext(ctx).GetMulti(ctx, wr.Event.Rooms, rooms)
	if err != nil {
		log.Printf("%v", err)
	}

	for i, room := range rooms {
		roomsMap[wr.Event.Rooms[i].ID] = *room
	}

	var buildingOrderMap = make(map[int64]int)
	for _, room := range wr.Event.Rooms {
		buildingKeyId := room.Parent.ID
		if _, present := buildingOrderMap[buildingKeyId]; !present {
			buildingOrderMap[buildingKeyId] = len(buildingOrderMap)
		}
	}

	var peopleToLookUp []*datastore.Key
	for _, booking := range bookings {
		peopleToLookUp = append(peopleToLookUp, booking.Roommates...)
	}

	personMap := make(map[int64]person.Person)
	var people = make([]*person.Person, len(peopleToLookUp))
	err = dsclient.FromContext(ctx).GetMulti(ctx, peopleToLookUp, people)
	if err != nil {
		log.Printf("%v", err)
	}

	for i, person := range people {
		personMap[peopleToLookUp[i].ID] = *person
	}

	var invitations []*Invitation
	q = datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		log.Printf("fetching invitations: %v", err)
	}

	personToInvitationMap := make(map[int64]int64)
	invitationMap := make(map[int64]Invitation)
	personToRsvpStatus := make(map[int64]invitation.RsvpStatus)
	for i, inv := range invitations {
		invitationMap[invitationKeys[i].ID] = *inv
		if inv.RsvpMap != nil {
			for p, r := range inv.RsvpMap {
				personToRsvpStatus[(*p).ID] = r
			}
		}
		for _, person := range inv.Invitees {
			personToInvitationMap[person.ID] = invitationKeys[i].ID
		}
	}

	shareBedBit := GetAllHousingPreferenceBooleans()[ShareBed].Bit

	type RealBooking struct {
		KeyString           string
		Room                housing.Room
		Building            *housing.Building
		Roommates           []person.Person
		ShowConvertToDouble bool
		FriSat              int
		PlusThurs           int
		AddThurs            []bool
		Cost                float64
		CostString          string
		Reserved            bool
	}

	buildingsMap := getBuildingMapForVenue(ctx, wr.Event.VenueKey())
	// doesn't deal with consolidating partitioned rooms
	var realBookingsByBuilding = make([][]RealBooking, len(buildingOrderMap))
	var totalCostForEveryone float64
	for i, booking := range bookings {
		people := make([]person.Person, len(booking.Roommates))

		FridaySaturday := 0
		PlusThursday := 0
		addThurs := make([]bool, len(booking.Roommates))
		doubleBedNeeded := false // for now don't deal with more than one double needed

		for i, person := range booking.Roommates {
			people[i] = personMap[person.ID]
			inv := invitationMap[personToInvitationMap[person.ID]]
			doubleBedNeeded = doubleBedNeeded || (inv.HousingPreferenceBooleans&shareBedBit == shareBedBit)
			rsvpStatus := personToRsvpStatus[person.ID]
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

		room := roomsMap[booking.Room.ID]
		buildingId := booking.Room.Parent.ID
		building := buildingsMap[buildingId]
		// for now? ignore case where they want a double bed and aren't getting it
		showConvertToDouble := doubleBedNeeded

		if doubleBedNeeded && (((building.Properties | room.Properties) & shareBedBit) == shareBedBit) {
			for _, bed := range room.Beds {
				if bed == housing.Double || bed == housing.Queen || bed == housing.King {
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
			Room:                roomsMap[booking.Room.ID],
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
		log.Printf("%v", err)
	}
}

func handleSaveReservations(ctx context.Context, wr WrappedRequest) {
	wr.Request.ParseForm()

	bookingToBooked := make(map[int64]bool)

	for k, v := range wr.Request.PostForm {
		log.Printf("key: %v", k)
		if v[0] == "" {
			continue
		}
		if k[0:8] == "booking_" {
			bookingKey, _ := datastore.DecodeKey(string(k[8:]))
			bookingToBooked[bookingKey.ID] = true
		}
	}

	var bookings []*Booking
	q := datastore.NewQuery("Booking").Ancestor(wr.EventKey)
	bookingKeys, _ := dsclient.FromContext(ctx).GetAll(ctx, q, &bookings)

	for i, booking := range bookings {
		if _, present := bookingToBooked[bookingKeys[i].ID]; present {
			booking.Reserved = true
		} else {
			booking.Reserved = false
		}
	}

	dsclient.FromContext(ctx).PutMulti(ctx, bookingKeys, bookings)

	http.Redirect(wr.ResponseWriter, wr.Request, "admin", http.StatusSeeOther)
}

func handleFoodReport(ctx context.Context, wr WrappedRequest) {
	currentEventKey := wr.EventKey

	allRsvpStatuses := invitation.GetAllRsvpStatuses()
	totalRestrictions := len(person.GetAllFoodRestrictionTags())

	counts := make([]int, totalRestrictions)
	personToRestrictions := make(map[int64][]bool)
	var people []person.Person

	var invitations []*Invitation
	q := datastore.NewQuery("Invitation").FilterField("Event", "=", currentEventKey)
	_, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		log.Printf("fetching invitations: %v", err)
	}

	for _, inv := range invitations {

		for p, s := range inv.RsvpMap {

			status := allRsvpStatuses[s]
			if status.Attending {
				var per person.Person
				dsclient.FromContext(ctx).Get(ctx, p, &per)
				people = append(people, per)
				restrictionsForPerson := make([]bool, totalRestrictions)
				for _, restriction := range per.FoodRestrictions {
					counts[restriction]++
					restrictionsForPerson[restriction] = true
				}
				personToRestrictions[p.ID] = restrictionsForPerson
			}
		}

	}

	sort.Slice(people, func(a, b int) bool { return person.SortByLastFirstName(people[a], people[b]) })

	tpl := template.Must(template.New("").ParseFiles("templates/main.html", "templates/foodReport.html"))
	data := wr.MakeTemplateData(map[string]interface{}{
		"AllRestrictions":      person.GetAllFoodRestrictionTags(),
		"Counts":               counts,
		"People":               people,
		"PersonToRestrictions": personToRestrictions,
	})

	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "foodReport.html", data); err != nil {
		log.Printf("%v", err)
	}
}

func handleRidesReport(ctx context.Context, wr WrappedRequest) {
	currentEventKey := wr.EventKey

	allRsvpStatuses := invitation.GetAllRsvpStatuses()

	type CarRequest struct {
		People               []*person.Person
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
	q := datastore.NewQuery("Invitation").FilterField("Event", "=", currentEventKey)
	_, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		log.Printf("fetching invitations: %v", err)
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

		var people = make([]*person.Person, len(personKeys))
		err = dsclient.FromContext(ctx).GetMulti(ctx, personKeys, people)
		if err != nil {
			log.Printf("fetching people: %v", err)
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
		log.Printf("%v", err)
	}
}
