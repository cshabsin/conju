package event

import (
	"time"

	"github.com/cshabsin/conju/invitation"
	"google.golang.org/appengine/datastore"
)

// TODO: add object that's a map of string names to values and attach one to every event
type Event struct {
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
