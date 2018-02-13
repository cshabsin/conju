package conju

// TODO: move to "package models"?

import (
	"context"
	"time"

	"google.golang.org/appengine/datastore"
)

type Event struct {
	Name      string
	StartDate time.Time
	EndDate   time.Time
	Current   bool
}

func CurrentEventKey(ctx context.Context) *datastore.Key {
	// TODO: Get current event from somewhere in datastore instead of
	// hard-coding it in code.
	return datastore.NewKey(ctx, "Event", "psr2018", 0, nil)
}

func CreateOneOffEvent(ctx context.Context) error {
	e := Event{
		Name: "Purity Spring 2018",
		StartDate: time.Date(2018, 6, 8, 0, 0, 0, 0, time.Local),
		EndDate: time.Date(2018, 6, 11, 0, 0, 0, 0, time.Local),
		Current: true,
	}
	_, err := datastore.Put(ctx, CurrentEventKey(ctx), &e)
	return err
}
