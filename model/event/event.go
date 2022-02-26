package event

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cshabsin/conju/invitation"
	"github.com/cshabsin/conju/model/venue"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

// TODO: add object that's a map of string names to values and attach one to every event
type eventDB struct {
	EventId               int // this can get deleted after all the data is imported
	Venue                 *datastore.Key
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
	EventId               int // this can get deleted after all the data is imported
	venueKey              *datastore.Key
	Venue                 *venue.Venue // only set after LoadVenue called.
	Name                  string
	ShortName             string
	StartDate             time.Time
	EndDate               time.Time
	RsvpStatuses          []invitation.RsvpStatus
	Rooms                 []*datastore.Key // TODO: replace with room
	Activities            []*datastore.Key // TODO: replace with activity
	InvitationClosingText string
	Current               bool
}

func (e *Event) LoadVenue(ctx context.Context) (*venue.Venue, error) {
	if e.Venue != nil {
		return e.Venue, nil
	}
	venue, err := venue.FromKey(ctx, e.venueKey)
	if err != nil {
		return nil, err
	}
	e.Venue = venue
	return e.Venue, nil
}

func (e *Event) EncodedKey() string {
	return e.Key.Encode()
}

func (e *Event) SetVenueKey(key *datastore.Key) {
	e.venueKey = key
	if e.Venue != nil && e.Venue.Key != e.venueKey {
		// change of venue, clear the old value
		e.Venue = nil
	}
}

func (e *Event) VenueKey() *datastore.Key {
	return e.venueKey
}

func (e *Event) ToDB() *eventDB {
	return &eventDB{
		EventId:               e.EventId,
		Venue:                 e.venueKey,
		Name:                  e.Name,
		ShortName:             e.ShortName,
		StartDate:             e.StartDate,
		EndDate:               e.EndDate,
		RsvpStatuses:          e.RsvpStatuses,
		Rooms:                 e.Rooms,
		Activities:            e.Activities,
		InvitationClosingText: e.InvitationClosingText,
		Current:               e.Current,
	}
}

func eventFromDB(ctx context.Context, key *datastore.Key, ev *eventDB) (*Event, error) {
	// TODO: get eventdb if called only with key
	return &Event{
		Key:                   key,
		EventId:               ev.EventId,
		venueKey:              ev.Venue,
		Name:                  ev.Name,
		ShortName:             ev.ShortName,
		StartDate:             ev.StartDate,
		EndDate:               ev.EndDate,
		RsvpStatuses:          ev.RsvpStatuses,
		Rooms:                 ev.Rooms,      // TODO: replace with keys
		Activities:            ev.Activities, // TODO: replace with keys
		InvitationClosingText: ev.InvitationClosingText,
		Current:               ev.Current,
	}, nil
}

func GetEvent(ctx context.Context, key *datastore.Key) (*Event, error) {
	var ev eventDB
	err := datastore.Get(ctx, key, &ev)
	if err != nil {
		return nil, err
	}
	return eventFromDB(ctx, key, &ev)
}

func PutEvent(ctx context.Context, ev *Event) error {
	if ev.Key == nil {
		ev.Key = datastore.NewIncompleteKey(ctx, "Event", nil)
	}
	_, err := datastore.Put(ctx, ev.Key, ev.ToDB())
	return err
}

func GetAllEvents(ctx context.Context) ([]*Event, error) {
	q := datastore.NewQuery("Event").Order("-StartDate")
	var allEventDBs []*eventDB
	eventKeys, err := q.GetAll(ctx, &allEventDBs)
	if err != nil {
		return nil, err
	}
	var allEvents []*Event
	for i := range allEventDBs {
		ev, err := eventFromDB(ctx, eventKeys[i], allEventDBs[i])
		if err != nil {
			return nil, fmt.Errorf("converting event: %w", err)
		}
		if _, err := ev.LoadVenue(ctx); err != nil {
			return nil, fmt.Errorf("loading venue: %w", err)
		}
		allEvents = append(allEvents, ev)
	}
	return allEvents, nil
}

func GetCurrentEvent(ctx context.Context) (*Event, error) {
	var keys []*datastore.Key
	var events []*eventDB
	q := datastore.NewQuery("Event").Filter("Current =", true)
	keys, err := q.GetAll(ctx, &events)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, errors.New("found no current event")
	}
	if len(keys) > 1 {
		return nil, fmt.Errorf("found more than one current event (%d)", len(keys))
	}
	return eventFromDB(ctx, keys[0], events[0])
}

func GetEventForHost(ctx context.Context, host string, e **Event, key **datastore.Key) (bool, error) {
	// TODO: generalize this for multiple hostnames/events.
	var shortname string
	if host == "psr2019.shabsin.com" {
		shortname = "PSR2019"
	} else if host == "psr2021.shabsin.com" {
		shortname = "PSR2021"
	} else {
		return false, nil
	}

	var keys []*datastore.Key
	var eventDBs []*eventDB
	q := datastore.NewQuery("Event").Filter("ShortName =", shortname)
	keys, err := q.GetAll(ctx, &eventDBs)
	if err != nil {
		log.Errorf(ctx, "Error querying for %s(url) event: %v", shortname, err)
		return false, nil
	}
	if len(keys) == 0 {
		log.Errorf(ctx, "Found no %s(url) event", shortname)
		return false, nil
	}
	if len(keys) > 1 {
		log.Errorf(ctx, "Found more than one %s(url) event (%d)", shortname, len(keys))
		return false, nil
	}
	ev, err := eventFromDB(ctx, keys[0], eventDBs[0])
	if err != nil {
		return false, err
	}
	*e = ev
	*key = keys[0]
	return true, nil
}
