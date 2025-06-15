package conju

import (
	"context"
	"log"
	"math"

	"cloud.google.com/go/datastore"
	"github.com/cshabsin/conju/conju/dsclient"
	"github.com/cshabsin/conju/invitation"
	"github.com/cshabsin/conju/model/housing"
	"github.com/cshabsin/conju/model/person"
)

// Booking holds the booking info and is kept in the datastore.
type Booking struct {
	Event    *datastore.Key
	Room     *datastore.Key
	Reserved bool

	Roommates []*datastore.Key
}

// InviteeRoomBookings holds the info for a given room's people.
// TODO: should this be renamed? Why is "Invitee" in the name?
type InviteeRoomBookings struct {
	Building            *housing.Building
	Room                *housing.Room
	Roommates           []*person.Person // People from this invitation.
	RoomSharers         []*person.Person // People from outside the invitation.
	ShowConvertToDouble bool
	ReservationMade     bool
}

// BuildingRoom holds a room of a building, for use as a key in a amp.
type BuildingRoom struct {
	Room     *housing.Room
	Building *housing.Building
}

// InviteeBookingsMap maps rooms to the InviteeRoomBookings that holds info
// about the people in the room.
type InviteeBookingsMap map[BuildingRoom]InviteeRoomBookings

func (ibm InviteeBookingsMap) IsKingPine() bool {
	for k := range ibm {
		if k.Building.Name == "Upper King Pine" {
			return true
		}
		if k.Building.Name == "Lower King Pine" {
			return true
		}
	}
	return false
}

// RoomingAndCostInfo contains the cost info for the people in one invitation.
type RoomingAndCostInfo struct {
	Invitation      *Invitation
	InviteeBookings InviteeBookingsMap
	Attendees       map[int64]*person.Person
	OrderedInvitees []*person.Person
	PersonToCost    map[*person.Person]float64
	TotalCost       float64
	IsThuFriSat     bool
}

// IsPaid returns true if the invitation's "received pay" is enough to consider the total cost paid off.
func (r RoomingAndCostInfo) IsPaid() bool {
	return r.TotalCost-r.Invitation.ReceivedPay < 0.05
}

func (r RoomingAndCostInfo) Unreserved() []BuildingRoom {
	var unreserved []BuildingRoom
	for _, booking := range r.InviteeBookings {
		if !booking.ReservationMade {
			unreserved = append(unreserved, BuildingRoom{booking.Room, booking.Building})
		}
	}
	return unreserved
}

// for now we're lazy and just return ThuFriSat if any status in the room is ThuFriSat; otherwise we assume FriSat in the template.
func (r RoomingAndCostInfo) ThuFriSat() bool {
	return r.IsThuFriSat
	// for _, status := range r.Invitation.RsvpMap {
	// 	if status == ThuFriSat {
	// 		return true
	// 	}
	// }
	// return false
}

func getRoomingInfo(ctx context.Context, wr WrappedRequest, invitationKey *datastore.Key) *RoomingAndCostInfo {
	// Load the invitation.
	var invitation Invitation
	err := dsclient.FromContext(ctx).Get(ctx, invitationKey, &invitation)
	if err != nil {
		log.Printf("Error retrieving invitation: %v", err)
	}
	return getRoomingInfoWithInvitation(ctx, wr, &invitation, invitationKey)
}

func getRoomingInfoWithInvitation(ctx context.Context, wr WrappedRequest, inv *Invitation,
	invitationKey *datastore.Key) *RoomingAndCostInfo {
	bookingInfo := wr.GetBookingInfo(ctx)

	// Construct set of Booking ids that contain any people in the invitation.
	bookingSet := make(map[int64]bool)
	for _, person := range inv.Invitees {
		if bookingID, ok := bookingInfo.PersonToBookingMap[person.ID]; ok {
			bookingSet[bookingID] = true
		}
	}

	if len(bookingSet) == 0 {
		return nil
	}

	var roomKeys []*datastore.Key
	var bookingsForInvitation []Booking
	for bookingID := range bookingSet {
		booking := bookingInfo.BookingKeyMap[bookingID]
		bookingsForInvitation = append(bookingsForInvitation, booking)
		roomKeys = append(roomKeys, booking.Room)
	}

	rooms := make([]*housing.Room, len(roomKeys))
	err := dsclient.FromContext(ctx).GetMulti(ctx, roomKeys, rooms)
	if err != nil {
		log.Printf("fetching rooms: %v", err)
	}

	// Map room ID -> Room
	roomsMap := make(map[int64]*housing.Room)
	for i, room := range rooms {
		roomsMap[roomKeys[i].ID] = room
	}

	var peopleToLookUp []*datastore.Key
	for _, booking := range bookingsForInvitation {
		peopleToLookUp = append(peopleToLookUp, booking.Roommates...)
	}

	personMap := make(map[int64]*person.Person)
	people := make([]*person.Person, len(peopleToLookUp))
	err = dsclient.FromContext(ctx).GetMulti(ctx, peopleToLookUp, people)
	if err != nil {
		log.Printf("fetching people: %v", err)
	}

	for i, person := range people {
		personMap[peopleToLookUp[i].ID] = person
	}

	var invitations []*Invitation
	q := datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		log.Printf("fetching invitations: %v", err)
	}

	personToRsvp := make(map[int64]invitation.RsvpStatus)
	personToInvitationMap := make(map[int64]int64)
	invitationMap := make(map[int64]*Invitation)
	for i, inv := range invitations {
		invitationMap[invitationKeys[i].ID] = inv
		for person, rsvp := range inv.RsvpMap {
			personToInvitationMap[person.ID] = invitationKeys[i].ID
			personToRsvp[person.ID] = rsvp
		}
	}
	shareBedBit := GetAllHousingPreferenceBooleans()[ShareBed].Bit

	wr.Event.LoadVenue(ctx)
	buildingsMap := getBuildingMapForVenue(ctx, wr.Event.Venue.Key)
	allInviteeBookings := make(map[int64]InviteeBookingsMap)
	personToCost := make(map[*person.Person]float64)
	isThuFriSat := false
	for _, booking := range bookingsForInvitation {
		room := roomsMap[booking.Room.ID]
		buildingID := booking.Room.Parent.ID
		building := buildingsMap[buildingID]
		buildingRoom := BuildingRoom{room, building}

		// Figure out if anyone's invitation signals need for a double bed.
		doubleBedNeeded := false
		for _, person := range booking.Roommates {
			invitation, ok := invitationMap[personToInvitationMap[person.ID]]
			if !ok {
				log.Printf("no invitation for person %v", person)
				break
			}
			doubleBedNeeded = doubleBedNeeded || (invitation.HousingPreferenceBooleans&shareBedBit == shareBedBit)
		}

		// Figure out if we need them to tell PSR to convert twin beds to double.
		showConvertToDouble := doubleBedNeeded
		if doubleBedNeeded && (((building.Properties | room.Properties) & shareBedBit) == shareBedBit) {
			for _, bed := range room.Beds {
				if bed == housing.Double || bed == housing.Queen || bed == housing.King {
					showConvertToDouble = false
					break
				}
			}
		}

		FridaySaturday := 0
		PlusThursday := 0
		addThurs := make([]bool, len(booking.Roommates))

		for i, per := range booking.Roommates {

			roommateInvitation := personToInvitationMap[per.ID]
			rsvpStatus := personToRsvp[per.ID]
			p := personMap[per.ID]

			if !p.IsBabyAtTime(wr.Event.StartDate) {
				if rsvpStatus == invitation.FriSat {
					FridaySaturday++
				}
				if rsvpStatus == invitation.ThuFriSat {
					FridaySaturday++
					PlusThursday++
					addThurs[i] = true
					isThuFriSat = true
				}
			}

			inviteeBookings, ok := allInviteeBookings[roommateInvitation]
			if !ok {
				inviteeBookings = make(InviteeBookingsMap)
				allInviteeBookings[roommateInvitation] = inviteeBookings
			}
			_, found := inviteeBookings[buildingRoom]
			if !found {
				roommates := make([]*person.Person, 0)
				roomSharers := make([]*person.Person, 0)
				for _, maybeRoommate := range booking.Roommates {
					maybeRoommatePerson := personMap[maybeRoommate.ID]
					if personToInvitationMap[maybeRoommate.ID] == roommateInvitation {
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

			p := personMap[person.ID]
			if p.IsBabyAtTime(wr.Event.StartDate) {
				personToCost[p] = 0
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
			personToCost[p] = costForPerson
		}

	}

	inviteePersonToCost := make(map[*person.Person]float64)
	var orderedInvitees []*person.Person
	var totalCost float64
	for _, invitee := range inv.Invitees {
		person := personMap[invitee.ID]
		if person == nil {
			continue
		}
		orderedInvitees = append(orderedInvitees, person)
		inviteePersonToCost[person] = personToCost[person]
		totalCost += personToCost[person]
	}

	return &RoomingAndCostInfo{
		Invitation:      inv,
		InviteeBookings: allInviteeBookings[invitationKey.ID],
		Attendees:       personMap,
		OrderedInvitees: orderedInvitees,
		PersonToCost:    inviteePersonToCost,
		TotalCost:       totalCost,
		IsThuFriSat:     isThuFriSat,
	}
}
