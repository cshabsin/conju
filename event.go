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
	EventId   int // this can get deleted after all the data is imported
	Name      string
	ShortName string
	StartDate time.Time
	EndDate   time.Time
	Current   bool
}

/*
func CurrentEventKey(ctx context.Context) (*datastore.Key, error) {
	ce_key := datastore.NewKey(ctx, "CurrentEvent", "current_event", 0, nil)
	var ce CurrentEvent
	err := datastore.Get(ctx, ce_key, &ce)
	if err != nil {
		k, err := CreateDefaultEvent(ctx)
		if err != nil {
			return nil, err
		}
		ce := CurrentEvent{k}
		datastore.Put(ctx, ce_key, &ce)
		return k, nil
	}
	return ce.Key, nil
}
*/
func CreateEvent(ctx context.Context, id int, name string, shortName string, startDate time.Time, endDate time.Time, current bool) (*datastore.Key, error) {
	e := Event{
		EventId:   id,
		Name:      name,
		ShortName: shortName,
		StartDate: startDate,
		EndDate:   endDate,
		Current:   current,
	}
	return datastore.Put(ctx, datastore.NewIncompleteKey(
		ctx, "Event", nil), &e)
}

/*
func CreateDefaultEvent(ctx context.Context) (*datastore.Key, error) {
	e := Event{
		Name:      "Purity Spring 2018",
		StartDate: time.Date(2018, 6, 8, 0, 0, 0, 0, time.Local),
		EndDate:   time.Date(2018, 6, 11, 0, 0, 0, 0, time.Local),
		Current:   true,
	}
	return datastore.Put(ctx, datastore.NewIncompleteKey(
		ctx, "Event", nil), &e)
}
*/
// Sets up Event in the WrappedRequest.
func EventGetter(wr *WrappedRequest) error {
	ctx := wr.Context
	encoded_key := wr.Values["event"]

	var key *datastore.Key = nil
	var err error
	var e *Event

	if encoded_key != nil {
		key, _ = datastore.DecodeKey(encoded_key.(string))
		err = datastore.Get(wr.Context, key, e)
	}

	var keys []*datastore.Key
	var events []*Event
	if e == nil {
		q := datastore.NewQuery("Event").Filter("Current =", true)
		keys, err = q.GetAll(ctx, &events)
		if len(keys) > 1 {
			log.Infof(ctx, "found %d current events", len(keys))
			return err

		}
		e = events[0]
		key = keys[0]
	}

	wr.Event = e
	//wr.EventKey = key.Encode()
	wr.SetSessionValue("EventKey", key.Encode())
	wr.SetSessionValue("Event", e)
	wr.SaveSession()

	return nil
}
