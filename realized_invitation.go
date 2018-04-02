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
	Housing                   HousingPreferenceInfo
	HousingPreferenceBooleans int
	HousingNotes              string
	Activities                []ActivityWithKey
	ActivitiesMap             map[string](map[string]ActivityRanking)
	ActivitiesLeadersMap      map[string](map[string]bool)
	Driving                   DrivingPreferenceInfo
	Parking                   ParkingTypeInfo
	LeaveFrom                 string
	LeaveTime                 string
	TravelNotes               string
	AdditionalPassengers      string
	OtherInfo                 string
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

	var activities []ActivityWithKey
	for _, activityKey := range event.Activities {
		var activity Activity
		datastore.Get(ctx, activityKey, &activity)
		encodedKey := activityKey.Encode()
		activities = append(activities, ActivityWithKey{Activity: activity, EncodedKey: encodedKey})
	}

	realizedActivityMap := make(map[string](map[string]ActivityRanking))
	for p, m := range invitation.ActivityMap {

		personMap := make(map[string]ActivityRanking)
		for a, r := range m {
			personMap[a.Encode()] = r
		}

		realizedActivityMap[p.Encode()] = personMap
	}
	realizedActivityLeadersMap := make(map[string](map[string]bool))
	for p, m := range invitation.ActivityLeaderMap {

		personMap := make(map[string]bool)
		for a, r := range m {
			personMap[a.Encode()] = r
		}

		realizedActivityLeadersMap[p.Encode()] = personMap
	}

	realizedInvitation := RealizedInvitation{
		EncodedKey:                invitationKey.Encode(),
		Invitees:                  invitees,
		Event:                     event,
		RsvpMap:                   realizedRsvpMap,
		Activities:                activities,
		ActivitiesMap:             realizedActivityMap,
		ActivitiesLeadersMap:      realizedActivityLeadersMap,
		Housing:                   GetAllHousingPreferences()[invitation.Housing],
		HousingNotes:              invitation.HousingNotes,
		HousingPreferenceBooleans: invitation.HousingPreferenceBooleans,
		Driving:                   GetAllDrivingPreferences()[invitation.Driving],
		Parking:                   GetAllParkingTypes()[invitation.Parking],
		LeaveFrom:                 invitation.LeaveFrom,
		LeaveTime:                 invitation.LeaveTime,
		AdditionalPassengers:      invitation.AdditionalPassengers,
		TravelNotes:               invitation.TravelNotes,
		OtherInfo:                 invitation.OtherInfo,
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

func RealInvHasHousingPreference(inv RealizedInvitation, preference HousingPreferenceBooleanInfo) bool {
	return (inv.HousingPreferenceBooleans & preference.Bit) > 0
}
