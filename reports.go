package conju

import (
	"fmt"
	"html/template"
	"sort"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

func handleReports(wr WrappedRequest) {
	var tpl = template.Must(template.ParseFiles("templates/main.html", "templates/reports.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "reports.html", wr.TemplateData); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}

func handleRsvpReport(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	currentEventKeyEncoded := wr.Values["EventKey"].(string)
	currentEventKey, _ := datastore.DecodeKey(currentEventKeyEncoded)

	var invitations []*Invitation
	q := datastore.NewQuery("Invitation").Filter("Event =", currentEventKey)
	_, err := q.GetAll(ctx, &invitations)
	if err != nil {
		log.Errorf(ctx, "fetching invitations: %v", err)
	}

	allRsvpMap := make(map[RsvpStatus][][]Person)
	var allNoRsvp [][]Person

	for _, invitation := range invitations {

		rsvpMap, noRsvp := invitation.ClusterByRsvp(ctx)

		for r, p := range rsvpMap {
			listOfLists := allRsvpMap[r]
			if listOfLists == nil {
				listOfLists = make([][]Person, 0)
			}
			listOfLists = append(listOfLists, p)
			allRsvpMap[r] = listOfLists
		}
		if len(noRsvp) > 0 {
			allNoRsvp = append(allNoRsvp, noRsvp)
		}
	}

	statusOrder := []RsvpStatus{ThuFriSat, FriSat, Maybe, No}

	tpl := template.Must(template.New("").ParseFiles("templates/main.html", "templates/rsvpReport.html"))
	data := wr.MakeTemplateData(map[string]interface{}{
		"RsvpMap":         allRsvpMap,
		"NoRsvp":          allNoRsvp,
		"StatusOrder":     statusOrder,
		"AllRsvpStatuses": GetAllRsvpStatuses(),
	})
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "rsvpReport.html", data); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}

func handleActivitiesReport(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	currentEventKeyEncoded := wr.Values["EventKey"].(string)
	currentEventKey, _ := datastore.DecodeKey(currentEventKeyEncoded)

	var invitations []*Invitation
	q := datastore.NewQuery("Invitation").Filter("Event =", currentEventKey)
	_, err := q.GetAll(ctx, &invitations)
	if err != nil {
		log.Errorf(ctx, "fetching invitations: %v", err)
	}

	allRsvpStatuses := GetAllRsvpStatuses()

	var activities = make([]*Activity, len(wr.Event.Activities))
	err = datastore.GetMulti(ctx, wr.Event.Activities, activities)
	if err != nil {
		log.Errorf(ctx, "fetching activities: %v", err)
	}

	keysToActivities := make(map[datastore.Key]Activity)
	activityKeys := make([]datastore.Key, len(wr.Event.Activities))
	for i, key := range wr.Event.Activities {
		keysToActivities[*key] = *activities[i]
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
	_, _ = q.GetAll(ctx, &bookings)

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
	personToRsvpStatus := make(map[int64]RsvpStatus)
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
		Room                Room
		Building            Building
		Roommates           []Person
		ShowConvertToDouble bool
		FriSat              int
		PlusThurs           int
		AddThurs            []bool
		Cost                float64
		CostString          string
	}

	buildingsMap := getBuildingMapForVenue(wr.Context, wr.Event.Venue)
	// doesn't deal with consolidating partitioned rooms
	var realBookingsByBuilding = make([][]RealBooking, len(buildingOrderMap))
	for _, booking := range bookings {
		people := make([]Person, len(booking.Roommates))

		FridaySaturday := 0
		PlusThursday := 0
		addThurs := make([]bool, len(booking.Roommates))
		doubleBedNeeded := false // for now don't deal with more than one double needed

		for i, person := range booking.Roommates {
			people[i] = personMap[person.IntID()]
			invitation := invitationMap[personToInvitationMap[person.IntID()]]
			doubleBedNeeded = doubleBedNeeded || (invitation.HousingPreferenceBooleans&shareBedBit == shareBedBit)
			rsvpStatus := personToRsvpStatus[person.IntID()]
			if people[i].IsBabyAtTime(wr.Event.StartDate) {
				continue
			}
			if rsvpStatus == FriSat {
				FridaySaturday++
			}
			if rsvpStatus == ThuFriSat {
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
		if FridaySaturday <= 4 {
			basePPCost = GetAllRsvpStatuses()[FriSat].BaseCost[FridaySaturday]
			basePPCostString = fmt.Sprintf("$%.2f", basePPCost)
		}
		addOnPPCost := float64(0)
		addOnPPCostString := "???"
		if PlusThursday <= 4 {
			addOnPPCost = GetAllRsvpStatuses()[ThuFriSat].AddOnCost[PlusThursday]
			addOnPPCostString = fmt.Sprintf("$%.2f", addOnPPCost)
		}

		addOnTotalString := ""
		if PlusThursday > 0 {
			addOnTotalString = fmt.Sprintf(" + %d * %s", PlusThursday, addOnPPCostString)
		}

		totalCost := float64(FridaySaturday)*basePPCost + float64(PlusThursday)*addOnPPCost
		totalCostString := fmt.Sprintf("$%.2f", totalCost)
		if FridaySaturday > 4 || PlusThursday > 4 {
			totalCostString = "???"
		}

		costEquationString := fmt.Sprintf("%d * %s %s = %s", FridaySaturday, basePPCostString, addOnTotalString, totalCostString)

		realBooking := RealBooking{
			Room:                roomsMap[booking.Room.IntID()],
			Building:            building,
			Roommates:           people,
			FriSat:              FridaySaturday,
			PlusThurs:           PlusThursday,
			AddThurs:            addThurs,
			ShowConvertToDouble: showConvertToDouble,
			Cost:                totalCost,
			CostString:          costEquationString,
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
		"BookingsByBuilding": realBookingsByBuilding,
	})
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "roomingReport.html", data); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}
