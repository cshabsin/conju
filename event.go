package conju

// TODO: move to "package models"?

import (
	"context"
	"time"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

type CurrentEvent struct {
	Key *datastore.Key
}

type Event struct {
	EventId      int // this can get deleted after all the data is imported
	Name         string
	ShortName    string
	StartDate    time.Time
	EndDate      time.Time
	RsvpStatuses []RsvpStatus
	Current      bool
}

func CreateEvent(ctx context.Context, id int, name string, shortName string, startDate time.Time, endDate time.Time, rsvpStatuses []RsvpStatus, current bool) (*datastore.Key, error) {
	e := Event{
		EventId:      id,
		Name:         name,
		ShortName:    shortName,
		StartDate:    startDate,
		EndDate:      endDate,
		RsvpStatuses: rsvpStatuses,
		Current:      current,
	}
	return datastore.Put(ctx, datastore.NewIncompleteKey(
		ctx, "Event", nil), &e)
}

// Sets up Event in the WrappedRequest.
func EventGetter(wr *WrappedRequest) error {
	if wr.hasRunEventGetter {
		return nil // Only retrieve once.
	}
	wr.hasRunEventGetter = true
	ctx := wr.Context
	key, err := wr.RetrieveKeyFromSession("event")
	if err != nil {
		// do something
	}
	var e *Event
	err = datastore.Get(wr.Context, key, e)
	if err == nil && e != nil {
		// We have retrieved the event successfully.
		return nil
	}

	var keys []*datastore.Key
	var events []*Event
	q := datastore.NewQuery("Event").Filter("Current =", true)
	keys, err = q.GetAll(ctx, &events)
	if len(keys) > 1 {
		log.Infof(ctx, "found %d current events", len(keys))
		return err

	}
	e = events[0]
	key = keys[0]

	wr.Event = e
	wr.EventKey = key
	wr.SetSessionValue("EventKey", key.Encode())
	wr.SetSessionValue("Event", e)
	wr.SaveSession()

	return nil
}
