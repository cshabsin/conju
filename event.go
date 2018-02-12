package conju

// TODO: move to "package models"?

import (
	"time"

	"google.golang.org/appengine/datastore"
)

type Event struct {
	Name        string
	StartDate   time.Time
	EndDate     time.Time
	PageSnippet *datastore.Key
}

