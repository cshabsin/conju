package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/cshabsin/conju/invitation"
)

type Event struct {
	Key                   *datastore.Key
	EventId               int // this can get deleted after all the data is imported
	venueKey              *datastore.Key
	Venue                 *datastore.Key // only set after LoadVenue called.
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

func (e Event) String() string {
	return fmt.Sprintf("[%02d:%s] %s", e.EventId, e.ShortName, e.Name)
}

func realMain(ctx context.Context) error {
	project := "useful-art-199822"
	datastoreClient, err := datastore.NewClient(ctx, project)
	if err != nil {
		return err
	}
	var events []*Event
	if _, err := datastoreClient.GetAll(ctx, datastore.NewQuery("Event"), &events); err != nil {
		return err
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].StartDate.Before(events[j].StartDate)
	})
	for _, e := range events {
		fmt.Println(e)
	}
	return nil
}

func main() {
	fmt.Println("yo")
	if err := realMain(context.Background()); err != nil {
		// In case of an auth error, try gcloud auth application-default login
		log.Fatal(err)
		os.Exit(1)
	}
}
