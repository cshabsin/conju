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
	EventId               int // this can get deleted after all the data is imported
	Name                  string
	ShortName             string
	StartDate             time.Time
	EndDate               time.Time
	RsvpStatuses          []RsvpStatus
	InvitationClosingText string
	Current               bool
}

func CreateEvent(ctx context.Context, id int, name string, shortName string, startDate time.Time, endDate time.Time, rsvpStatuses []RsvpStatus, invitationClosingText string, current bool) (*datastore.Key, error) {
	e := Event{
		EventId:               id,
		Name:                  name,
		ShortName:             shortName,
		StartDate:             startDate,
		EndDate:               endDate,
		RsvpStatuses:          rsvpStatuses,
		InvitationClosingText: invitationClosingText,
		Current:               current,
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
	key, err := wr.RetrieveKeyFromSession("EventKey")
	if err != nil {
		// TODO: do something
		return err
	}
	var e *Event
	err = datastore.Get(wr.Context, key, e)
	if err == nil && e != nil {
		// We have retrieved the event successfully.
		log.Infof(wr.Context, "retrieved event successfully.")
		return nil
	}

	var keys []*datastore.Key
	var events []*Event
	q := datastore.NewQuery("Event").Filter("Current =", true)
	keys, err = q.GetAll(wr.Context, &events)
	if len(keys) > 1 {
		log.Infof(wr.Context, "found %d current events", len(keys))
		return err
	}
	e = events[0]
	key = keys[0]

	wr.Event = e
	wr.EventKey = key
	wr.SetSessionValue("EventKey", key.Encode())
	wr.SaveSession()

	return nil
}
