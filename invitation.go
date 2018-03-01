package conju

// TODO: move to "package models"?

import (
	"google.golang.org/appengine/datastore"
)

type Invitation struct {
	Event    *datastore.Key   // Event
	Invitees []*datastore.Key // []Person
}
