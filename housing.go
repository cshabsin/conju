package conju

import (
	"context"
	"math"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

type Venue struct {
	Name          string
	ShortName     string
	ContactPerson string
	ContactPhone  string
	ContactEmail  string
	Website       string
}

type Building struct {
	Venue             *datastore.Key
	Name              string
	Code              string
	Properties        int
	FloorplanImageUrl string
}

type BedSize int

const (
	King = iota
	Queen
	Double
	Twin
	Cot
)

type Room struct {
	Building    *datastore.Key
	RoomNumber  int
	Description string
	Partition   string
	Properties  int
	Beds        []BedSize

	ImageTop    int
	ImageLeft   int
	ImageWidth  int
	ImageHeight int
}

type RealRoom struct {
	Room       Room
	Building   Building
	BedsString string
}

func (room Room) makeRealRoom(ctx context.Context) RealRoom {
	var building Building
	datastore.Get(ctx, room.Building, building)

	var realRoom RealRoom
	realRoom.Building = building
	realRoom.Room = room

	return realRoom
}

func (room RealRoom) AllProperties() int {
	return room.Building.Properties | room.Room.Properties
}

type Booking struct {
	Event    *datastore.Key
	Room     *datastore.Key
	Reserved bool

	Roommates []*datastore.Key
}

type InviteeRoomBookings struct {
	Building            *Building
	Room                *Room
	Roommates           []*Person // People from this invitation.
	RoomSharers         []*Person // People from outside the invitation.
	ShowConvertToDouble bool
	ReservationMade     bool
}

type BuildingRoom struct {
	Room     *Room
	Building *Building
}

type InviteeBookingsMap map[BuildingRoom]InviteeRoomBookings

type RoomingAndCostInfo struct {
	InviteeBookings InviteeBookingsMap
	OrderedInvitees []*Person
	PersonToCost    map[*Person]float64
	TotalCost       float64
}

func getRoomingInfo(wr WrappedRequest, invitationKey *datastore.Key) *RoomingAndCostInfo {

	ctx := wr.Context

	var invitation Invitation
	datastore.Get(wr.Context, invitationKey, &invitation)

	var bookings []Booking
	q := datastore.NewQuery("Booking").Ancestor(wr.EventKey)
	allBookingKeys, _ := q.GetAll(ctx, &bookings)

	bookingKeyToBookingMap := make(map[int64]Booking)
	personToBookingMap := make(map[int64]int64)
	for b, booking := range bookings {
		bookingKeyToBookingMap[allBookingKeys[b].IntID()] = booking
		for _, person := range booking.Roommates {
			personToBookingMap[person.IntID()] = allBookingKeys[b].IntID()
		}
	}

	bookingSet := make(map[int64]bool)
	for _, person := range invitation.Invitees {
		if bookingId, ok := personToBookingMap[person.IntID()]; ok {

			bookingSet[bookingId] = true
		}
	}

	if len(bookingSet) == 0 {
		return nil
	}

	var roomKeys []*datastore.Key
	var bookingsForInvitation []Booking
	for bookingId, _ := range bookingSet {
		booking := bookingKeyToBookingMap[bookingId]
		bookingsForInvitation = append(bookingsForInvitation, booking)
		roomKeys = append(roomKeys, booking.Room)
	}

	var rooms = make([]*Room, len(roomKeys))
	err := datastore.GetMulti(ctx, roomKeys, rooms)
	if err != nil {
		log.Errorf(ctx, "fetching rooms: %v", err)
	}

	// Map room ID -> Room
	roomsMap := make(map[int64]*Room)
	for i, room := range rooms {
		roomsMap[roomKeys[i].IntID()] = room
	}

	var peopleToLookUp []*datastore.Key
	for _, booking := range bookingsForInvitation {
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

	personToRsvp := make(map[int64]RsvpStatus)
	personToInvitationMap := make(map[int64]int64)
	invitationMap := make(map[int64]*Invitation)
	for i, inv := range invitations {
		invitationMap[invitationKeys[i].IntID()] = inv
		for person, rsvp := range inv.RsvpMap {
			personToInvitationMap[person.IntID()] = invitationKeys[i].IntID()
			personToRsvp[person.IntID()] = rsvp
		}
	}
	shareBedBit := GetAllHousingPreferenceBooleans()[ShareBed].Bit

	buildingsMap := getBuildingMapForVenue(ctx, wr.Event.Venue)
	allInviteeBookings := make(map[int64]InviteeBookingsMap)
	personToCost := make(map[*Person]float64)
	for _, booking := range bookingsForInvitation {
		room := roomsMap[booking.Room.IntID()]
		buildingId := booking.Room.Parent().IntID()
		building := buildingsMap[buildingId]
		buildingRoom := BuildingRoom{room, building}

		// Figure out if anyone's invitation signals need for a double bed.
		doubleBedNeeded := false
		for _, person := range booking.Roommates {
			invitation := invitationMap[personToInvitationMap[person.IntID()]]
			doubleBedNeeded = doubleBedNeeded || (invitation.HousingPreferenceBooleans&shareBedBit == shareBedBit)
		}

		// Figure out if we need them to tell PSR to convert twin beds to double.
		showConvertToDouble := doubleBedNeeded

		if doubleBedNeeded && (((building.Properties | room.Properties) & shareBedBit) == shareBedBit) {
			for _, bed := range room.Beds {
				if bed == Double || bed == Queen || bed == King {
					showConvertToDouble = false
					break
				}
			}
		}

		FridaySaturday := 0
		PlusThursday := 0
		addThurs := make([]bool, len(booking.Roommates))

		for i, person := range booking.Roommates {

			roommateInvitation := personToInvitationMap[person.IntID()]
			rsvpStatus := personToRsvp[person.IntID()]
			p := personMap[person.IntID()]

			if !p.IsBabyAtTime(wr.Event.StartDate) {
				if rsvpStatus == FriSat {
					FridaySaturday++
				}
				if rsvpStatus == ThuFriSat {
					FridaySaturday++
					PlusThursday++
					addThurs[i] = true
				}
			}

			inviteeBookings, found := allInviteeBookings[roommateInvitation]
			if !found {
				inviteeBookings = make(InviteeBookingsMap)
				allInviteeBookings[roommateInvitation] = inviteeBookings
			}
			_, found = inviteeBookings[buildingRoom]
			if !found {
				roommates := make([]*Person, 0)
				roomSharers := make([]*Person, 0)
				for _, maybeRoommate := range booking.Roommates {
					maybeRoommatePerson := personMap[maybeRoommate.IntID()]
					if personToInvitationMap[maybeRoommate.IntID()] == roommateInvitation {
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

		for i, person := range booking.Roommates {

			p := personMap[person.IntID()]
			if p.IsBabyAtTime(wr.Event.StartDate) {
				personToCost[p] = 0
				continue
			}
			costForPerson := float64(0)
			if FridaySaturday <= 5 {
				costForPerson = GetAllRsvpStatuses()[FriSat].BaseCost[FridaySaturday]
			}

			if addThurs[i] && PlusThursday <= 5 {
				costForPerson += GetAllRsvpStatuses()[ThuFriSat].AddOnCost[PlusThursday]
			}
			costForPerson = math.Floor(costForPerson*100) / 100
			personToCost[p] = costForPerson
		}

	}

	inviteePersonToCost := make(map[*Person]float64)
	var orderedInvitees []*Person
	var totalCost float64
	for _, invitee := range invitation.Invitees {
		person := personMap[invitee.IntID()]
		orderedInvitees = append(orderedInvitees, person)
		inviteePersonToCost[person] = personToCost[person]
		totalCost += personToCost[person]
	}

	return &RoomingAndCostInfo{
		InviteeBookings: allInviteeBookings[invitationKey.IntID()],
		OrderedInvitees: orderedInvitees,
		PersonToCost:    inviteePersonToCost,
		TotalCost:       totalCost,
	}
}
