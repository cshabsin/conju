package legacy

import (
	"fmt"
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
