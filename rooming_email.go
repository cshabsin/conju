package conju

import (
	"html/template"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

func handleSendRoomingEmail(wr WrappedRequest) {
	// Cribbed heavily from handleRoomingReport
	ctx := wr.Context

	var bookings []Booking
	q := datastore.NewQuery("Booking").Ancestor(wr.EventKey)
	_, err := q.GetAll(ctx, &bookings)
	if err != nil {
		log.Errorf(ctx, "fetching bookings: %v", err)
	}

	var rooms = make([]*Room, len(wr.Event.Rooms))
	err = datastore.GetMulti(ctx, wr.Event.Rooms, rooms)
	if err != nil {
		log.Errorf(ctx, "fetching rooms: %v", err)
	}

	// Map room ID -> Room
	roomsMap := make(map[int64]*Room)
	for i, room := range rooms {
		roomsMap[wr.Event.Rooms[i].IntID()] = room
	}

	var peopleToLookUp []*datastore.Key
	for _, booking := range bookings {
		peopleToLookUp = append(peopleToLookUp, booking.Roommates...)
	}

	personMap := make(map[int64]*Person)
	var people = make([]*Person, len(peopleToLookUp))
	err = datastore.GetMulti(ctx, peopleToLookUp, people)
	if err != nil {
		log.Errorf(ctx, "fetching people: %v", err)
	}

	for i, person := range people {
		personMap[peopleToLookUp[i].IntID()] = person
	}

	var invitations []*Invitation
	q = datastore.NewQuery("Invitation").Filter("Event =", wr.EventKey)
	invitationKeys, err := q.GetAll(ctx, &invitations)
	if err != nil {
		log.Errorf(ctx, "fetching invitations: %v", err)
	}

	personToInvitationMap := make(map[int64]int64)
	invitationMap := make(map[int64]*Invitation)
	for i, inv := range invitations {
		invitationMap[invitationKeys[i].IntID()] = inv
		for _, person := range inv.Invitees {
			personToInvitationMap[person.IntID()] = invitationKeys[i].IntID()
		}
	}
	// shareBedBit := GetAllHousingPreferenceBooleans()[ShareBed].Bit

	type BuildingRoom struct {
		Room     *Room
		Building *Building
	}
	type InviteeRoomBookings struct {
		Building            *Building
		Room                *Room
		Roommates           []*Person // People from this invitation.
		RoomSharers         []*Person // People from outside the invitation.
		ShowConvertToDouble bool
		ReservationMade     bool
	}
	type InviteeBookings map[BuildingRoom]InviteeRoomBookings

	buildingsMap := getBuildingMapForVenue(wr.Context, wr.Event.Venue)
	allInviteeBookings := make(map[int64]InviteeBookings)
	for _, booking := range bookings {
		room := roomsMap[booking.Room.IntID()]
		buildingId := booking.Room.Parent().IntID()
		building := buildingsMap[buildingId]
		buildingRoom := BuildingRoom{room, building}
		showConvertToDouble := false // TODO: Need to still implement convert to double.

		for _, person := range booking.Roommates {
			invitation := personToInvitationMap[person.IntID()]

			inviteeBookings, found := allInviteeBookings[invitation]
			if !found {
				inviteeBookings = make(InviteeBookings)
				allInviteeBookings[invitation] = inviteeBookings
			}
			_, found = inviteeBookings[buildingRoom]
			if !found {
				roommates := make([]*Person, 0)
				roomSharers := make([]*Person, 0)
				for _, maybeRoommate := range booking.Roommates {
					maybeRoommatePerson := personMap[maybeRoommate.IntID()]
					if personToInvitationMap[maybeRoommate.IntID()] == invitation {
						roommates = append(roommates, maybeRoommatePerson)
					} else {
						roomSharers = append(roomSharers, maybeRoommatePerson)
					}
				}
				inviteeBookings[buildingRoom] = InviteeRoomBookings{
					Building:            building,
					Room:                room,
					Roommates:           roommates,
					RoomSharers:         roomSharers,
					ShowConvertToDouble: showConvertToDouble,
					ReservationMade:     booking.Reserved,
				}
			}
		}
	}

	functionMap := template.FuncMap{
		"HasHousingPreference":        RealInvHasHousingPreference,
		"PronounString":               GetPronouns,
		"CollectiveAddressFirstNames": CollectiveAddressFirstNames,
		"SharerName":                  MakeSharerName,
		"DerefPeople":                 DerefPeople,
	}
	tpl := template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/PSR2018/email/rooming.html"))
	for invitation, bookings := range allInviteeBookings {
		// invitation is ID from key.
		ri := makeRealizedInvitation(ctx, *datastore.NewKey(ctx, "Invitation", "", invitation, nil), *invitationMap[invitation])
		unreserved := make([]BuildingRoom, 0)
		for _, booking := range bookings {
			if !booking.ReservationMade {
				unreserved = append(unreserved, BuildingRoom{booking.Room, booking.Building})
			}
		}
		data := wr.MakeTemplateData(map[string]interface{}{
			"Invitation":      ri,
			"InviteeBookings": bookings,
			"LoginLink":       "login link here",
			"Unreserved":      unreserved,
		})
		if err := tpl.ExecuteTemplate(wr.ResponseWriter, "rooming_html", data); err != nil {
			log.Errorf(wr.Context, "%v", err)
		}
	}
}

func MakeSharerName(p *Person) string {
	s := p.FullName()
	if p.Email != "" {
		s = s + "(" + p.Email + ")"
	}
	return s
}

func DerefPeople(people []*Person) []Person {
	dp := make([]Person, len(people))
	for i, p := range people {
		dp[i] = *p
	}
	return dp
}
