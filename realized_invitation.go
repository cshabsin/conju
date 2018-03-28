package conju

import (
	"context"

	"google.golang.org/appengine/datastore"
)

type RealizedInvitation struct {
	EncodedKey                string
	Invitees                  []PersonWithKey
	Event                     Event
	RsvpMap                   map[string]RsvpStatusInfo
	Housing                   HousingPreference
	HousingPreferenceBooleans int
	HousingNotes              string
	Driving                   DrivingPreference
	LeaveFrom                 string
	LeaveTime                 string
	TravelNotes               string
	AdditionalPassengers      string
}

func makeRealizedInvitation(ctx context.Context, invitationKey datastore.Key, invitation Invitation, getEvent bool) RealizedInvitation {
	personKeys := invitation.Invitees
	var invitees []PersonWithKey
	for _, personKey := range personKeys {
		var person Person
		datastore.Get(ctx, personKey, &person)
		person.DatastoreKey = personKey
		personWithKey := PersonWithKey{
			Person: person,
			Key:    personKey.Encode(),
		}

		invitees = append(invitees, personWithKey)
	}

	var event Event

	if getEvent {
		datastore.Get(ctx, invitation.Event, &event)
	}

	allRsvpStatuses := GetAllRsvpStatuses()
	realizedRsvpMap := make(map[string]RsvpStatusInfo)

	for k, v := range invitation.RsvpMap {
		realizedRsvpMap[k.Encode()] = allRsvpStatuses[v]
	}

	realizedInvitation := RealizedInvitation{
		EncodedKey:                invitationKey.Encode(),
		Invitees:                  invitees,
		Event:                     event,
		RsvpMap:                   realizedRsvpMap,
		Housing:                   invitation.Housing,
		HousingNotes:              invitation.HousingNotes,
		HousingPreferenceBooleans: invitation.HousingPreferenceBooleans,
		Driving:                   invitation.Driving,
		LeaveFrom:                 invitation.LeaveFrom,
		LeaveTime:                 invitation.LeaveTime,
		AdditionalPassengers:      invitation.AdditionalPassengers,
		TravelNotes:               invitation.TravelNotes,
	}

	return realizedInvitation
}

func printInvitation(ctx context.Context, key datastore.Key, inv Invitation) string {
	real := makeRealizedInvitation(ctx, key, inv, true)
	toReturn := real.Event.ShortName + ": "
	for _, invitee := range real.Invitees {
		toReturn += invitee.Person.FullName() + " - "
		statusString := "???"
		status, exists := real.RsvpMap[invitee.Key]
		if exists {
			statusString = status.ShortDescription
		}
		toReturn += statusString + ", "
	}
	return toReturn
}

func makeRealizedInvitations(ctx context.Context, invitationKeys []*datastore.Key, invitations []*Invitation) []RealizedInvitation {
	realizedInvitations := make([]RealizedInvitation, len(invitations))
	for i := 0; i < len(invitations); i++ {
		realizedInvitations[i] = makeRealizedInvitation(ctx, *invitationKeys[i], *invitations[i], false)
	}
	return realizedInvitations
}
