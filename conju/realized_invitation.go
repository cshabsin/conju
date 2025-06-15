package conju

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/datastore"

	"github.com/cshabsin/conju/activity"
	"github.com/cshabsin/conju/conju/dsclient"
	"github.com/cshabsin/conju/invitation"
	"github.com/cshabsin/conju/model/event"
	"github.com/cshabsin/conju/model/person"
)

type RealizedInvitation struct {
	Invitation                *Invitation
	EncodedKey                string
	Invitees                  []person.PersonWithKey
	Event                     *event.Event
	RsvpMap                   map[string]invitation.RsvpStatusInfo
	Housing                   HousingPreferenceInfo
	HousingPreferenceBooleans int
	HousingNotes              string
	Activities                []activity.ActivityWithKey
	ActivitiesMap             map[string](map[string]ActivityRanking)
	ActivitiesLeadersMap      map[string](map[string]bool)
	Driving                   DrivingPreferenceInfo
	Parking                   ParkingTypeInfo
	LeaveFrom                 string
	LeaveTime                 string
	TravelNotes               string
	AdditionalPassengers      string
	ThursdayDinnerCount       int
	FridayLunch               bool
	FridayDinnerCount         int
	FridayIceCreamCount       int
	OtherInfo                 string
	LastUpdatedPerson         person.PersonWithKey
	LastUpdatedTimestamp      time.Time
	InviteePeople             []person.Person
	ReceivedPayDateStr        string
	Thursday                  bool
	COVIDAcked                bool
	Storyland                 bool
}

func (ri RealizedInvitation) GetPeopleComing() []person.Person {
	peopleComing := make([]person.Person, 0)
	for i, p := range ri.Invitees {
		if ri.RsvpMap[p.Key].Attending {
			peopleComing = append(peopleComing, ri.InviteePeople[i])
		}
	}
	return peopleComing
}

func makeRealizedInvitation(ctx context.Context, invitationKey *datastore.Key, inv *Invitation) RealizedInvitation {
	personKeys := inv.Invitees
	var inviteePeople []person.Person
	var invitees []person.PersonWithKey
	for _, personKey := range personKeys {
		var pers person.Person
		if err := dsclient.FromContext(ctx).Get(ctx, personKey, &pers); err != nil {
			log.Printf("Error retrieving person %v: %v", personKey, err)
			continue
		}
		pers.DatastoreKey = personKey
		personWithKey := person.PersonWithKey{
			Person: pers,
			Key:    personKey.Encode(),
		}

		invitees = append(invitees, personWithKey)
		inviteePeople = append(inviteePeople, pers)
	}

	var pers person.Person
	var lastUpdatedPerson person.PersonWithKey
	err := dsclient.FromContext(ctx).Get(ctx, inv.LastUpdatedPerson, &pers)
	if err != nil {
		//log.Printf( "%v", err)
	} else {
		pers.DatastoreKey = inv.LastUpdatedPerson
		lastUpdatedPerson = person.PersonWithKey{
			Person: pers,
			Key:    inv.LastUpdatedPerson.Encode(),
		}
	}

	event, err := event.GetEvent(ctx, inv.Event)
	if err != nil {
		log.Printf("GetEvent: %v", err)
	}

	allRsvpStatuses := invitation.GetAllRsvpStatuses()
	realizedRsvpMap := make(map[string]invitation.RsvpStatusInfo)
	thursday := false
	for k, v := range inv.RsvpMap {
		realizedRsvpMap[k.Encode()] = allRsvpStatuses[v]
		if v == invitation.ThuFriSat {
			thursday = true
		}
	}

	var activities []activity.ActivityWithKey
	for i, activityKey := range event.Activities {
		if activityKey == nil {
			log.Printf("nil activityKey in event %v (index %d) (list %v)", event, i, event.Activities)
		}
		var act activity.Activity
		dsclient.FromContext(ctx).Get(ctx, activityKey, &act)
		encodedKey := activityKey.Encode()
		activities = append(activities, activity.ActivityWithKey{Activity: act, EncodedKey: encodedKey})
	}

	realizedActivityMap := make(map[string](map[string]ActivityRanking))
	for p, m := range inv.ActivityMap {

		personMap := make(map[string]ActivityRanking)
		for a, r := range m {
			personMap[a.Encode()] = r
		}

		realizedActivityMap[p.Encode()] = personMap
	}
	realizedActivityLeadersMap := make(map[string](map[string]bool))
	for p, m := range inv.ActivityLeaderMap {

		personMap := make(map[string]bool)
		for a, r := range m {
			personMap[a.Encode()] = r
		}

		realizedActivityLeadersMap[p.Encode()] = personMap
	}

	realizedInvitation := RealizedInvitation{
		Invitation:                inv,
		EncodedKey:                invitationKey.Encode(),
		Invitees:                  invitees,
		InviteePeople:             inviteePeople,
		Event:                     event,
		RsvpMap:                   realizedRsvpMap,
		Activities:                activities,
		ActivitiesMap:             realizedActivityMap,
		ActivitiesLeadersMap:      realizedActivityLeadersMap,
		Housing:                   GetAllHousingPreferences()[inv.Housing],
		HousingNotes:              inv.HousingNotes,
		HousingPreferenceBooleans: inv.HousingPreferenceBooleans,
		Driving:                   GetAllDrivingPreferences()[inv.Driving],
		Parking:                   GetAllParkingTypes()[inv.Parking],
		LeaveFrom:                 inv.LeaveFrom,
		LeaveTime:                 inv.LeaveTime,
		AdditionalPassengers:      inv.AdditionalPassengers,
		TravelNotes:               inv.TravelNotes,
		ThursdayDinnerCount:       inv.ThursdayDinnerCount,
		FridayLunch:               inv.FridayLunch,
		FridayDinnerCount:         inv.FridayDinnerCount,
		FridayIceCreamCount:       inv.FridayIceCreamCount,
		OtherInfo:                 inv.OtherInfo,
		LastUpdatedPerson:         lastUpdatedPerson,
		LastUpdatedTimestamp:      inv.LastUpdatedTimestamp,
		ReceivedPayDateStr:        inv.ReceivedPayDate.Format("2006-01-02"),
		Thursday:                  thursday,
		COVIDAcked:                inv.COVIDAcked,
		Storyland:                 inv.Storyland,
	}

	return realizedInvitation
}

func printInvitation(ctx context.Context, key *datastore.Key, inv *Invitation) string {
	real := makeRealizedInvitation(ctx, key, inv)
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
		realizedInvitations[i] = makeRealizedInvitation(ctx, invitationKeys[i], invitations[i])
	}
	return realizedInvitations
}

func RealInvHasHousingPreference(inv RealizedInvitation, preference HousingPreferenceBooleanInfo) bool {
	return (inv.HousingPreferenceBooleans & preference.Bit) > 0
}
