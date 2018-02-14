package conju

// TODO: move to "package models"?

import (
	"google.golang.org/appengine/datastore"
)

type Invitation struct {
	Guest          []*datastore.Key  // Person
	InvitationCode string
}

