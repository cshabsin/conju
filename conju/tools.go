package conju

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"

	"cloud.google.com/go/datastore"

	"github.com/cshabsin/conju/conju/dsclient"
	"github.com/cshabsin/conju/invitation"
	"github.com/cshabsin/conju/model/housing"
	"github.com/cshabsin/conju/model/person"
)

func handleRoomingTool(ctx context.Context, wr WrappedRequest) {
	var bookings []Booking
	q := datastore.NewQuery("Booking").Ancestor(wr.EventKey)
	bookingKeys, _ := dsclient.FromContext(ctx).GetAll(ctx, q, &bookings)

	type BookingInfo struct {
		Booking    Booking
		RoomString string
	}

	var personToBooking = make(map[int64]int64)
	bookingInfos := make([]BookingInfo, len(bookings))
	var roomStringMap = make(map[int64]string)

	var invitationsToExplode []string

	wr.Event.LoadVenue(ctx)
	buildingsMap := getBuildingMapForVenue(ctx, wr.Event.Venue.Key)
	var buildingsInOrder []housing.Building
	var availableRooms []*housing.RealRoom
	var buildingsToRooms = make(map[housing.Building][]*housing.RealRoom)

	for _, room := range wr.Event.Rooms {
		var rm housing.Room
		if err := dsclient.FromContext(ctx).Get(ctx, room, &rm); err != nil {
			log.Printf("Reading room (id %s): %v", room.Encode(), err)
			continue
		}
		buildingKey := room.Parent
		building, ok := buildingsMap[buildingKey.ID]
		if !ok {
			log.Printf("building not found in buildingsMap for building %v", buildingKey)
			continue
		}
		if building == nil {
			log.Printf("nil building in buildingsMap for building %v", buildingKey)
			continue
		}
		if len(buildingsInOrder) == 0 || buildingsInOrder[len(buildingsInOrder)-1] != *building {
			buildingsInOrder = append(buildingsInOrder, *building)
		}

		bedstring := ""
		for _, bed := range rm.Beds {
			switch bed {
			case housing.King:
				bedstring += "K"
			case housing.Queen:
				bedstring += "Q"
			case housing.Double:
				bedstring += "D"
			case housing.Twin:
				bedstring += "T"
			case housing.Cot:
				bedstring += "C"

			}
		}
		realRoom := &housing.RealRoom{
			Room:       rm,
			Building:   *buildingsMap[buildingKey.ID],
			BedsString: bedstring,
		}

		buildingsToRooms[*building] = append(buildingsToRooms[*building], realRoom)

		availableRooms = append(availableRooms, realRoom)

		roomStr := building.Code + "_" + strconv.Itoa(rm.RoomNumber)
		if rm.Partition != "" {
			roomStr += "_" + rm.Partition
		}
		roomStringMap[room.ID] = roomStr
	}

	for i, booking := range bookings {
		bookingInfos[i] = BookingInfo{Booking: booking, RoomString: roomStringMap[booking.Room.ID]}
		for _, roommate := range booking.Roommates {
			personToBooking[roommate.ID] = bookingKeys[i].ID
		}
	}

	var invitations []*Invitation
	q = datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	_, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		log.Printf("fetching invitations: %v", err)
	}

	statusOrder := []invitation.RsvpStatus{invitation.FriSatSun, invitation.FriSat, invitation.SatSun, invitation.Maybe}
	adultPreferenceMask := GetAdultPreferenceMask()
	rsvpToGroupsMap := make(map[invitation.RsvpStatus][][]person.Person)
	var noRsvps [][]person.Person
	peopleToProperties := make(map[*datastore.Key]int)

	for _, invitation := range invitations {
		rsvpMap, noResponse := invitation.ClusterByRsvp(ctx)
		for _, s := range statusOrder {
			if peopleForRsvp, pr := rsvpMap[s]; pr {
				if listForRsvp, present := rsvpToGroupsMap[s]; present {
					listForRsvp = append(listForRsvp, peopleForRsvp)
					rsvpToGroupsMap[s] = listForRsvp
				} else {
					listForRsvp = [][]person.Person{}
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

					bookingId := personToBooking[person.DatastoreKey.ID]
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
		sort.Slice(v, func(a, b int) bool { return person.SortByFirstName(v[a][0], v[b][0]) })
	}

	sort.Slice(noRsvps, func(a, b int) bool { return person.SortByFirstName(noRsvps[a][0], noRsvps[b][0]) })

	tpl := template.Must(template.New("").ParseFiles("templates/main.html", "templates/roomingTool.html"))
	data := wr.MakeTemplateData(map[string]interface{}{
		"RsvpToGroupsMap":      rsvpToGroupsMap,
		"NoRsvps":              noRsvps,
		"StatusOrder":          statusOrder,
		"AllRsvpStatuses":      invitation.GetAllRsvpStatuses(),
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
		log.Printf("%v", err)
	}

}

func handleSaveRooming(ctx context.Context, wr WrappedRequest) {
	wr.Request.ParseForm()

	q := datastore.NewQuery("Booking").Ancestor(wr.EventKey).KeysOnly()
	bookingKeys, _ := dsclient.FromContext(ctx).GetAll(ctx, q, nil)
	// TODO: don't unilaterally delete all old bookings -- look for changes.
	// (Will matter when saving booked state.)
	dsclient.FromContext(ctx).DeleteMulti(ctx, bookingKeys)

	wr.Event.LoadVenue(ctx)
	buildingMap := getBuildingMapForVenue(ctx, wr.Event.Venue.Key)

	roomMap := make(map[string]*datastore.Key)
	roomingMap := make(map[string][]*datastore.Key)
	var rooms = make([]*housing.Room, len(wr.Event.Rooms))
	dsclient.FromContext(ctx).GetMulti(ctx, wr.Event.Rooms, rooms)

	for i, room := range rooms {
		str := buildingMap[room.Building.ID].Code + "_" + strconv.Itoa(room.RoomNumber)
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
	q = datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		log.Printf("fetching invitations: %v", err)
	}

	personToInvitationMap := make(map[int64]int64)
	personToInvitationIndexMap := make(map[int64]int)
	invitationMap := make(map[int64]Invitation)
	var peopleToLookUp []*datastore.Key
	for i, inv := range invitations {
		invitationMap[invitationKeys[i].ID] = *inv
		for p, person := range inv.Invitees {
			peopleToLookUp = append(peopleToLookUp, person)
			personToInvitationMap[person.ID] = invitationKeys[i].ID
			personToInvitationIndexMap[person.ID] = p
		}

	}
	var people = make([]*person.Person, len(peopleToLookUp))
	dsclient.FromContext(ctx).GetMulti(ctx, peopleToLookUp, people)
	personMap := make(map[int64]person.Person)
	for i, person := range people {
		personMap[peopleToLookUp[i].ID] = *person
	}

	for rmStr, people := range roomingMap {
		if len(people) == 0 {
			continue
		}

		countByInvitation := make(map[int64]int)
		peopleForRoom := make(map[int64]bool)
		for _, person := range people {
			countByInvitation[personToInvitationMap[person.ID]]++
			peopleForRoom[person.ID] = true
		}

		sort.Slice(people, func(a, b int) bool {
			invA := personToInvitationMap[people[a].ID]
			invB := personToInvitationMap[people[b].ID]
			if invA == invB {
				return personToInvitationIndexMap[people[a].ID] < personToInvitationIndexMap[people[b].ID]
			}
			invCountA := countByInvitation[invA]
			invCountB := countByInvitation[invB]
			if invCountA != invCountB {
				return invCountA > invCountB
			}
			// really we want to sort by first person on each invitation, close enough for now.
			return person.SortByLastFirstName(personMap[people[a].ID], personMap[people[b].ID])
		})

		booking := Booking{Event: wr.EventKey, Room: roomMap[rmStr], Roommates: people}
		dsclient.FromContext(ctx).Put(ctx, datastore.IncompleteKey("Booking", wr.EventKey), &booking)
	}

	http.Redirect(wr.ResponseWriter, wr.Request, "rooming", http.StatusSeeOther)
}

func getBuildingMapForVenue(ctx context.Context, venueKey *datastore.Key) map[int64]*housing.Building {
	buildingsMap := make(map[int64]*housing.Building)
	var buildings []*housing.Building
	q := datastore.NewQuery("Building").Ancestor(venueKey)
	keys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &buildings)

	if err != nil {
		log.Printf("%v", err)
	}
	for i, buildingKey := range keys {
		buildingsMap[buildingKey.ID] = buildings[i]
	}
	return buildingsMap
}
