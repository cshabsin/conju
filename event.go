package conju

// TODO: move to "package models"?

import (
	"context"
	"time"

	"google.golang.org/appengine/datastore"
)

type CurrentEvent struct {
	Key *datastore.Key
}

type Event struct {
	Name      string
	StartDate time.Time
	EndDate   time.Time
	Current   bool
}

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

func cleanUp(ctx context.Context) error {
	q := datastore.NewQuery("CurrentEvent")
	for t := q.Run(ctx); ; {
		var ce CurrentEvent
		key, err := t.Next(&ce)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return err
		}
		err = datastore.Delete(ctx, key)
		if err != nil {
			return err
		}
	}
	return nil
}

