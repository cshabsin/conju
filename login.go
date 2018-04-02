package conju

import (
	"fmt"
	"math/rand"

	"google.golang.org/appengine/datastore"
)

// A LoginCode is a secret string we send to users as part of their
// Login link. It's stored as a string field in the Person object.
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

type LoginInfo struct {
	InvitationKey *datastore.Key
	*Invitation
	PersonKey *datastore.Key
	*Person
}

const loginPage = "/login"

// When a user navigates to the login link and provides the given code
// string, the system validates the login code against the Person
// table, and either puts the login code into the session, or writes
// an error.
func handleLogin(wr WrappedRequest) {
	url_q := wr.URL.Query()
	lc, ok := url_q["loginCode"]
	if !ok {
		wr.ResponseWriter.Write([]byte("Please use the link from your email to log in."))
		return
	}
	count, err := datastore.NewQuery("Person").Filter("LoginCode =", lc[0]).Count(wr.Context)
	if err != nil {
		wr.ResponseWriter.Write([]byte("Login not recognized."))
		return
	}
	if count == 0 {
		wr.ResponseWriter.Write([]byte("Login not recognized."))
		return
	}
	wr.SetSessionValue("code", lc[0])
	wr.SaveSession()
	wr.ResponseWriter.Write([]byte(fmt.Sprintf("Got loginCode: %s\n", lc[0])))
}

// LoginGetter validates the login code from the session, looking up
// the Person with the matching code. Then it finds the Invitation in
// the current Event (per the WrappedRequest field Event) that
// references that person. It stores the Person and Invitation (key
// and value) into the WrappedRequest's LoginInfo.  This getter will
// redirect to the login screen if the LoginCode is not found in the
// database.
//
// If EventGetter has not been called, LoginGetter calls it.
func LoginGetter(wr *WrappedRequest) error {
	if !wr.hasRunEventGetter {
		err := EventGetter(wr)
		if err != nil {
			return err
		}
	}
	code, ok := wr.Values["code"].(string)
	if !ok {
		return RedirectError{loginPage}
	}
	var people []Person
	peopleKeys, err := datastore.NewQuery("Person").Filter("LoginCode =", code).GetAll(wr.Context, &people)
	if err != nil {
		return err
	}
	if len(people) == 0 {
		return RedirectError{loginPage + "?message=Person not found for loginCode."}
	} else if len(people) > 1 {
		return RedirectError{loginPage + "?message=DB Error: loginCode collision."}
	}

	var invitations []Invitation
	invitationKeys, err := datastore.NewQuery("Invitation").
		Filter("Invitees =", peopleKeys[0]).
		Filter("Event =", wr.EventKey).
		GetAll(wr.Context, &invitations)
	if err != nil {
		return err
	}
	if len(invitations) == 0 {
		return RedirectError{loginPage + "?message=No invitation found for currently selected event."}
	} else if len(invitations) > 1 {
		return RedirectError{loginPage + "?message=DB Error: multiple invitations found."}
	}

	wr.LoginInfo = &LoginInfo{invitationKeys[0], &invitations[0], peopleKeys[0], &people[0]}
	return nil
}

// Simple URL handler that prints out the invitation retrieved by
// LoginGetter, for testing.
func checkLogin(wr WrappedRequest) {
	wr.ResponseWriter.Write([]byte(fmt.Sprintf("Invitation: %s", printInvitation(wr.Context, *wr.LoginInfo.InvitationKey, *wr.LoginInfo.Invitation))))
}
