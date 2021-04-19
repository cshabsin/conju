// Package poll defines a poll data type that can collect users' responses to questions.
package poll

import (
	"context"
	"fmt"

	"google.golang.org/appengine/datastore"
)

// Answer defines an answer to the one question we have, "how crazy is this?" (from PSR 2021).
// Should be generalized someday.
type Answer struct {
	Invitation *datastore.Key
	Question   int // always 1 for now
	Rating     int
}

// GetAnswer returns the key and data for an invitation's answer to the poll. Returns nil if the
// user hasn't answered yet.
func GetAnswer(ctx context.Context, invitationKey *datastore.Key) (*datastore.Key, *Answer, error) {
	q := datastore.NewQuery("Answer").Filter("Invitation=", invitationKey).Filter("Question=", 1)
	if q == nil {
		return nil, nil, fmt.Errorf("nil query for invitation %q", invitationKey.String())
	}
	var answers []*Answer
	keys, err := q.GetAll(ctx, &answers)
	if err != nil {
		return nil, nil, err
	}
	if len(keys) == 0 {
		return nil, nil, nil
	}
	if len(keys) > 1 {
		return nil, nil, fmt.Errorf("more than one poll response for invitation %q", invitationKey.String())
	}
	return keys[0], answers[0], nil
}

// SetAnswer sets an invitation's poll answer. If the caller has access to the answerKey from a previous
// GetAnswer call this can be done more efficiently.
func SetAnswer(ctx context.Context, invitationKey *datastore.Key, answerKey *datastore.Key, rating int) (*datastore.Key, error) {
	answer := &Answer{
		Invitation: invitationKey,
		Question:   1,
		Rating:     rating,
	}
	if answerKey != nil {
		return datastore.Put(ctx, answerKey, answer)
	}
	return datastore.Put(ctx, datastore.NewIncompleteKey(ctx, "Answer", nil), answer)
}
