package conju

import (
	"context"
	"errors"
	"math/rand"

	"google.golang.org/appengine/datastore"
)

// A LoginCode is a secret string we send to users as part of their
// Login link. The LoginCode datastore entry is keyed under the
// StringId "login/<codestring>", with a parent id of an Event
// ID. That way, an invitation link can continue to work even when
// the string value is copied from a previous event.
//
// When a user navigates to the login link and provides the given code
// string, the system puts the code string, the Person Key, and the
// Invitation Key into the associated Session. In subsequent requests,
// Handlers can apply one of two Getter functions to act on these fields:
//
//  - the LoginGetter retrieves the Person and Invitation objects
//    associated with the session and stores pointers to them in the
//    WrappedRequest.
//
//  - the LoginValidateGetter simply verifies that the LoginCode is
//    still present in the datastore.
//
// Either of these getters will redirect to the login screen if the
// LoginCode has been removed from the database.
type LoginCode struct {
	Code       string
	Invitation *datastore.Key
	Person     *datastore.Key
}

const loginCodeLength = 12

func randomLoginCodeString() string {
	b := make([]rune, loginCodeLength)
	for i := range b {
		r := rand.Intn(36)
		if r < 10 {
			// 0..9
			b[i] = int32(r) + 48
		} else {
			// A..Z ((r - 10) + 65)
			b[i] = int32(r) + 55
		}
	}
	return string(b)
}

func CreateLoginCode(ctx context.Context, event *datastore.Key,
	invitation *datastore.Key, person *datastore.Key) (string, *datastore.Key, error) {
	// TODO: figure out whether the person already has a login code for this invitation/event.
	// TODO: guard against overwriting existing login codes (in case of a duplicate random value)
	lcs := randomLoginCodeString()
	incomplete_key := datastore.NewKey(ctx, "LoginCode", lcs, 0, event)
	lc := LoginCode{lcs, invitation, person}
	key, err := datastore.Put(ctx, incomplete_key, lc)
	if err != nil {
		return "", nil, err
	}
	return lcs, key, nil
}

type LoginInfo struct {
	InvitationKey *datastore.Key
	*Invitation
	PersonKey *datastore.Key
	*Person
}

func LoginGetter(wr *WrappedRequest) error {
	if !wr.hasRunEventGetter {
		return errors.New("LoginGetter called without EventGetter")
	}
	code, ok := wr.Values["code"].(string)
	if !ok {
		return RedirectError{"/login"}
	}
	var lc *LoginCode
	err := datastore.Get(wr.Context, datastore.NewKey(wr.Context, "LoginCode", code, 0, wr.EventKey), lc)
	if err != nil {
		return err
	}
	var invitation *Invitation
	err = datastore.Get(wr.Context, lc.Invitation, invitation)
	if err != nil {
		return err
	}

	var person *Person
	err = datastore.Get(wr.Context, lc.Person, person)
	if err != nil {
		return err
	}

	wr.LoginInfo = &LoginInfo{lc.Invitation, invitation, lc.Person, person}
	return nil
}