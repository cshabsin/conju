package event

import (
	"time"

	"github.com/cshabsin/conju/conju"
	"github.com/cshabsin/conju/invitation"
	"github.com/cshabsin/conju/model/activity"
	"github.com/cshabsin/conju/model/housing"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

// TODO: add object that's a map of string names to values and attach one to every event
type eventDB struct {
	EventId               int            // this can get deleted after all the data is imported
	Venue                 *datastore.Key // TODO(cshabsin): split eventDB from Event and make this an actual object
	Name                  string
	ShortName             string
	StartDate             time.Time
	EndDate               time.Time
	RsvpStatuses          []invitation.RsvpStatus
	Rooms                 []*datastore.Key
	Activities            []*datastore.Key
	InvitationClosingText string
	Current               bool
}

type Event struct {
	Key                   *datastore.Key
	Venue                 *housing.Venue
	Name                  string
	ShortName             string
	StartDate             time.Time
	EndDate               time.Time
	RsvpStatuses          []invitation.RsvpStatus
	Rooms                 []*housing.Room
	Activities            []*activity.Activity
	InvitationClosingText string
	Current               bool
}

// GetEventForHost returns a default event based on the host in the request, if it can be inferred.
func GetEventForHost(wr *conju.WrappedRequest) (*Event, error) {
	host := wr.GetHost()
	// TODO: generalize this for multiple hostnames/events.
	var shortname string
	if host == "psr2019.shabsin.com" {
		shortname = "PSR2019"
	} else if host == "psr2021.shabsin.com" {
		shortname = "PSR2021"
	} else {
		return nil, nil
	}

	var keys []*datastore.Key
	var events []*Event
	q := datastore.NewQuery("Event").Filter("ShortName =", shortname)
	keys, err := q.GetAll(wr.Context, &events)
	if err != nil {
		log.Errorf(wr.Context, "Error querying for %s(url) event: %v", shortname, err)
		return false, nil
	}
	if len(keys) == 0 {
		log.Errorf(wr.Context, "Found no %s(url) event", shortname)
		return false, nil
	}
	if len(keys) > 1 {
		log.Errorf(wr.Context, "Found more than one %s(url) event (%d)", shortname, len(keys))
		return false, nil
	}
	*e = events[0]
	*key = keys[0]
	return true, nil
}
