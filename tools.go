package conju

import (
	"context"
	"html/template"
	"net/http"
	"sort"
	"strconv"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

func handleRoomingTool(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)

	var bookings []Booking
	q := datastore.NewQuery("Booking").Ancestor(wr.EventKey)
	bookingKeys, _ := q.GetAll(ctx, &bookings)

	type BookingInfo struct {
		Booking    Booking
		RoomString string
	}

	var personToBooking = make(map[int64]int64)
	bookingInfos := make([]BookingInfo, len(bookings))
	var roomStringMap = make(map[int64]string)

	var invitationsToExplode []string

	buildingsMap := getBuildingMapForVenue(wr.Context, wr.Event.Venue)
	var buildingsInOrder []Building
	var availableRooms []RealRoom
	var buildingsToRooms = make(map[Building][]RealRoom)

	for i, room := range wr.Event.Rooms {
		var rm Room
		datastore.Get(ctx, room, &rm)
		buildingKey := room.Parent()
		building := buildingsMap[buildingKey.IntID()]
		if i == 0 || buildingsInOrder[len(buildingsInOrder)-1] != *building {
			buildingsInOrder = append(buildingsInOrder, *building)
		}

		bedstring := ""
		for _, bed := range rm.Beds {
			switch bed {
			case King:
				bedstring += "K"
			case Queen:
				bedstring += "Q"
			case Double:
				bedstring += "D"
			case Twin:
				bedstring += "T"
			case Cot:
				bedstring += "C"

			}
		}
		realRoom := RealRoom{
			Room:       rm,
			Building:   *buildingsMap[buildingKey.IntID()],
			BedsString: bedstring,
		}

		roomsForBuilding := buildingsToRooms[*building]

		if roomsForBuilding == nil {
			roomsForBuilding = make([]RealRoom, 0)

		}
		roomsForBuilding = append(roomsForBuilding, realRoom)
		buildingsToRooms[*building] = roomsForBuilding

		availableRooms = append(availableRooms, realRoom)

		roomStr := building.Code + "_" + strconv.Itoa(rm.RoomNumber)
		if rm.Partition != "" {
			roomStr += "_" + rm.Partition
		}
		roomStringMap[room.IntID()] = roomStr
	}

	for i, booking := range bookings {
		bookingInfos[i] = BookingInfo{Booking: booking, RoomString: roomStringMap[booking.Room.IntID()]}
		for _, roommate := range booking.Roommates {
			personToBooking[roommate.IntID()] = bookingKeys[i].IntID()
		}
	}

	var invitations []*Invitation
	q = datastore.NewQuery("Invitation").Filter("Event =", wr.EventKey)
	_, err := q.GetAll(ctx, &invitations)
	if err != nil {
		log.Errorf(ctx, "fetching invitations: %v", err)
	}

	statusOrder := []RsvpStatus{ThuFriSat, FriSat, Maybe}
	adultPreferenceMask := GetAdultPreferenceMask()
	rsvpToGroupsMap := make(map[RsvpStatus][][]Person)
	var noRsvps [][]Person
	peopleToProperties := make(map[*datastore.Key]int)

	for _, invitation := range invitations {
		rsvpMap, noResponse := invitation.ClusterByRsvp(ctx)
		for _, s := range statusOrder {
			if peopleForRsvp, pr := rsvpMap[s]; pr {
				if listForRsvp, present := rsvpToGroupsMap[s]; present {
					listForRsvp = append(listForRsvp, peopleForRsvp)
					rsvpToGroupsMap[s] = listForRsvp
				} else {
					listForRsvp = [][]Person{}
					listForRsvp = append(listForRsvp, peopleForRsvp)
					rsvpToGroupsMap[s] = listForRsvp
				}
				initialBookingId := int64(0)
				foundExploder := false
				for i, person := range peopleForRsvp {
					hpb := invitation.HousingPreferenceBooleans
					if person.IsAdultAtTime(wr.Event.StartDate) {
						hpb |= adultPreferenceMask
					}
					peopleToProperties[person.DatastoreKey] = hpb

					bookingId := personToBooking[person.DatastoreKey.IntID()]
					if i == 0 {
						initialBookingId = bookingId
					} else {
						if !foundExploder && initialBookingId != bookingId {
							invitationsToExplode = append(invitationsToExplode, person.DatastoreKey.Encode())
							foundExploder = true
						}
					}
				}

			}
		}
		if len(noResponse) > 0 {
			noRsvps = append(noRsvps, noResponse)
		}
	}

	for _, v := range rsvpToGroupsMap {
		sort.Slice(v, func(a, b int) bool { return SortByFirstName(v[a][0], v[b][0]) })
	}

	sort.Slice(noRsvps, func(a, b int) bool { return SortByFirstName(noRsvps[a][0], noRsvps[b][0]) })

	tpl := template.Must(template.New("").ParseFiles("templates/main.html", "templates/roomingTool.html"))
	data := wr.MakeTemplateData(map[string]interface{}{
		"RsvpToGroupsMap":      rsvpToGroupsMap,
		"NoRsvps":              noRsvps,
		"StatusOrder":          statusOrder,
		"AllRsvpStatuses":      GetAllRsvpStatuses(),
		"AvailableRooms":       availableRooms,
		"BuildingsToRooms":     buildingsToRooms,
		"BuildingsInOrder":     buildingsInOrder,
		"PeopleToProperties":   peopleToProperties,
		"DesiredMask":          GetPreferenceTypeMask(Desired),
		"AcceptableMask":       GetPreferenceTypeMask(Acceptable),
		"BookingInfos":         bookingInfos,
		"InvitationsToExplode": invitationsToExplode,
	})
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "roomingTool.html", data); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}

}

func handleSaveRooming(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	wr.Request.ParseForm()

	q := datastore.NewQuery("Booking").Ancestor(wr.EventKey).KeysOnly()
	bookingKeys, _ := q.GetAll(ctx, nil)
	// TODO: don't unilaterally delete all old bookings -- look for changes.
	// (Will matter when saving booked state.)
	datastore.DeleteMulti(ctx, bookingKeys)

	buildingMap := getBuildingMapForVenue(ctx, wr.Event.Venue)

	roomMap := make(map[string]*datastore.Key)
	roomingMap := make(map[string][]*datastore.Key)
	var rooms = make([]*Room, len(wr.Event.Rooms))
	datastore.GetMulti(ctx, wr.Event.Rooms, rooms)

	for i, room := range rooms {
		str := buildingMap[room.Building.IntID()].Code + "_" + strconv.Itoa(room.RoomNumber)
		if room.Partition != "" {
			str += "_" + room.Partition
		}
		roomMap[str] = wr.Event.Rooms[i]
		roomingMap[str] = make([]*datastore.Key, 0)
	}

	for k, v := range wr.Request.PostForm {
		if v[0] == "" {
			continue
		}
		if k[0:12] == "roomingSlot_" {
			personKey, _ := datastore.DecodeKey(string(k[12:]))
			roommates := roomingMap[v[0]]
			roomingMap[v[0]] = append(roommates, personKey)
		}
	}

	var invitations []*Invitation
	q = datastore.NewQuery("Invitation").Filter("Event =", wr.EventKey)
	invitationKeys, err := q.GetAll(ctx, &invitations)
	if err != nil {
		log.Errorf(ctx, "fetching invitations: %v", err)
	}

	personToInvitationMap := make(map[int64]int64)
	personToInvitationIndexMap := make(map[int64]int)
	invitationMap := make(map[int64]Invitation)
	var peopleToLookUp []*datastore.Key
	for i, inv := range invitations {
		invitationMap[invitationKeys[i].IntID()] = *inv
		for p, person := range inv.Invitees {
			peopleToLookUp = append(peopleToLookUp, person)
			personToInvitationMap[person.IntID()] = invitationKeys[i].IntID()
			personToInvitationIndexMap[person.IntID()] = p
		}

	}
	var people = make([]*Person, len(peopleToLookUp))
	datastore.GetMulti(ctx, peopleToLookUp, people)
	personMap := make(map[int64]Person)
	for i, person := range people {
		personMap[peopleToLookUp[i].IntID()] = *person
	}

	for rmStr, people := range roomingMap {
		if len(people) == 0 {
			continue
		}

		countByInvitation := make(map[int64]int)
		peopleForRoom := make(map[int64]bool)
		for _, person := range people {
			countByInvitation[personToInvitationMap[person.IntID()]]++
			peopleForRoom[person.IntID()] = true
		}

		sort.Slice(people, func(a, b int) bool {
			invA := personToInvitationMap[people[a].IntID()]
			invB := personToInvitationMap[people[b].IntID()]
			if invA == invB {
				return personToInvitationIndexMap[people[a].IntID()] < personToInvitationIndexMap[people[b].IntID()]
			}
			invCountA := countByInvitation[invA]
			invCountB := countByInvitation[invB]
			if invCountA != invCountB {
				return invCountA > invCountB
			}
			// really we want to sort by first person on each invitation, close enough for now.
			return SortByLastFirstName(personMap[people[a].IntID()], personMap[people[b].IntID()])
		})

		booking := Booking{Event: wr.EventKey, Room: roomMap[rmStr], Roommates: people}
		datastore.Put(ctx, datastore.NewIncompleteKey(ctx, "Booking", wr.EventKey), &booking)
	}

	http.Redirect(wr.ResponseWriter, wr.Request, "rooming", http.StatusSeeOther)
}

func getBuildingMapForVenue(ctx context.Context, venueKey *datastore.Key) map[int64]*Building {
	buildingsMap := make(map[int64]*Building)
	var buildings []*Building
	q := datastore.NewQuery("Building").Ancestor(venueKey)
	keys, err := q.GetAll(ctx, &buildings)

	if err != nil {
		log.Infof(ctx, "%v", err)
	}
	for i, buildingKey := range keys {
		buildingsMap[buildingKey.IntID()] = buildings[i]
	}
	return buildingsMap
}
