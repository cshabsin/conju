package conju

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"cloud.google.com/go/datastore"
	"github.com/gin-gonic/gin"
	"github.com/cshabsin/conju/conju/dsclient"
	"github.com/cshabsin/conju/model/person"
	"google.golang.org/appengine/v2/user"
)

type LoginInfo struct {
	InvitationKey *datastore.Key
	*Invitation
	PersonKey *datastore.Key
	*person.Person
}

const loginErrorPage = "/loginError"
const resentInvitationPage = "/resentInvitation"

func handleLogin(urlTarget string) gin.HandlerFunc {
	return func(c *gin.Context) {
		wr, ok := c.MustGet("wrappedRequest").(*WrappedRequest)
		if !ok {
			log.Printf("could not get wrapped request from context")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		handleLoginInner(c, wr, urlTarget)
	}
}

// When a user navigates to the login link and provides the given code
// string, the system validates the login code against the Person
// table, and either puts the login code into the session, or writes
// an error. On error, we display an error page with help. On success,
// we redirect to urlTarget.
func handleLoginInner(c *gin.Context, wr *WrappedRequest, urlTarget string) {
	ctx := c.Request.Context()
	dsclient := dsclient.FromContext(ctx)

	// TODO(cshabsin): Read "message" CGI arg if present and
	// display it. Prettify this page in general, using templates.
	url_q := wr.Request.URL.Query()
	lc, ok := url_q["loginCode"]
	if !ok {
		c.Redirect(http.StatusFound, loginErrorPage+
			"?message=Login is required for this section of our site.  Please use the link from your email to log in.")
		return
	}
	var people []person.Person
	q := datastore.NewQuery("Person").FilterField("LoginCode", "=", lc[0])
	peopleKeys, err := dsclient.GetAll(ctx, q, &people)
	if err != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s?message=DB error looking you up: %v", loginErrorPage, err))
		return
	}
	if len(peopleKeys) == 0 {
		c.Redirect(http.StatusFound, loginErrorPage+
			"?message=Login not recognized.")
		return
	}
	if len(peopleKeys) > 1 {
		c.Redirect(http.StatusFound, loginErrorPage+
			"?message=DB Error: multiple invitations found.")
	}
	wr.SetSessionValue("code", lc[0])
	wr.SetSessionValue("person", peopleKeys[0].Encode())
	wr.SaveSession()
	c.Redirect(http.StatusFound, urlTarget)
}

func getPersonFromEncodedKey(ctx context.Context, wr *WrappedRequest) (*datastore.Key, *person.Person, error) {
	dsclient := dsclient.FromContext(ctx)

	log.Printf("getPersonFromEncodedKey")
	personKeyEncoded, ok := wr.Values["person"].(string)
	if !ok {
		log.Printf("person cookie not set")
		return nil, nil, errors.New("person cookie not set")
	}
	personKey, err := datastore.DecodeKey(personKeyEncoded)
	if err != nil {
		log.Printf("person key decode error: %v", err)
		return nil, nil, err
	}
	pers := person.Person{}
	err = dsclient.Get(ctx, personKey, &pers)
	if err != nil {
		log.Printf("person get error: %v", err)
		return nil, nil, err
	}
	return personKey, &pers, nil
}

func getPersonFromLoggedInUser(ctx context.Context, wr *WrappedRequest) (*datastore.Key, *person.Person, error) {
	dsclient := dsclient.FromContext(ctx)

	log.Printf("getPersonFromLoggedInUser")
	if wr.User == nil {
		log.Printf("not logged in")
		return nil, nil, errors.New("not logged in")
	}
	var people []*person.Person
	q := datastore.NewQuery("Person").FilterField("Email", "=", wr.User.Email)
	peopleKeys, err := dsclient.GetAll(ctx, q, &people)
	if err != nil {
		log.Printf("person lookup by email (%v) error: %v", wr.User.Email, err)
		return nil, nil, err
	}
	if len(people) > 1 {
		log.Printf("collision on email (%v)", wr.User.Email)
		// multiple people with the same email address, punt to code.
		return nil, nil, fmt.Errorf("multiple people with email address %v", wr.User.Email)
	}
	return peopleKeys[0], people[0], nil
}

func getPersonFromInvitationCode(ctx context.Context, wr *WrappedRequest) (*datastore.Key, *person.Person, error) {
	dsclient := dsclient.FromContext(ctx)

	log.Printf("getPersonFromInvitationCode")
	code, ok := wr.Values["code"].(string)
	if !ok {
		log.Printf("invitation code not set")
		return nil, nil, errors.New("invitation code not set")
	}
	var people []*person.Person
	q := datastore.NewQuery("Person").FilterField("LoginCode", "=", code)
	peopleKeys, err := dsclient.GetAll(ctx, q, &people)
	if err != nil {
		log.Printf("person lookup by login code (%v) error: %v", code, err)
		return nil, nil, err
	}
	if len(people) > 1 {
		log.Printf("collision on loginCode (%v)", code)
		return nil, nil, fmt.Errorf("loginCode collision: %q", code)
	}
	return peopleKeys[0], people[0], nil
}

func getPersonFromSession(ctx context.Context, wr *WrappedRequest) (*datastore.Key, *person.Person, bool, error) {
	key, person, err := getPersonFromEncodedKey(ctx, wr)
	if err == nil {
		return key, person, false, err
	}
	key, person, err = getPersonFromLoggedInUser(ctx, wr)
	if err == nil {
		return key, person, true, err
	}
	key, person, err = getPersonFromInvitationCode(ctx, wr)
	if err == nil {
		return key, person, true, err
	}
	return nil, nil, false, err
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
func PersonGetter(ctx context.Context, wr *WrappedRequest) error {
	if wr.LoginInfo != nil {
		return nil // This has already been run.
	}
	personKey, pers, writeToSession, err := getPersonFromSession(ctx, wr)
	if err != nil {
		log.Printf("error getting person: %v", err)
	}
	li := &LoginInfo{
		InvitationKey: nil,
		Invitation:    nil,
		PersonKey:     personKey,
		Person:        pers,
	}
	wr.LoginInfo = li
	wr.TemplateData["LoginInfo"] = li
	if writeToSession {
		wr.SetSessionValue("person", personKey.Encode())
		wr.SaveSession()
	}
	return nil
}

func InvitationGetter(ctx context.Context, wr *WrappedRequest) error {
	dsclient := dsclient.FromContext(ctx)

	if wr.LoginInfo == nil {
		if err := PersonGetter(ctx, wr); err != nil {
			log.Printf("couldn't get person: %v", err)
			return err
		}
	}
	if !wr.hasRunEventGetter {
		if err := EventGetter(ctx, wr); err != nil {
			log.Printf("couldn't get event: %v", err)
			return err
		}
	}
	if wr.Event == nil {
		log.Printf("nil event")
		// Do something.
	}
	if wr.LoginInfo.Person == nil {
		return &RedirectError{loginErrorPage + "?message=Please use the link from your invitation email to log in."}
	}
	var invitations []Invitation
	q := datastore.NewQuery("Invitation").
		FilterField("Invitees", "=", wr.LoginInfo.PersonKey).
		FilterField("Event", "=", wr.EventKey)
	invitationKeys, err := dsclient.GetAll(ctx, q, &invitations)
	if err != nil {
		return err
	}
	if len(invitations) == 0 {
		return &RedirectError{loginErrorPage + "?message=No invitation found for currently selected event"}
	} else if len(invitations) > 1 {
		return &RedirectError{loginErrorPage + "?message=DB Error: multiple invitations found."}
	}

	wr.LoginInfo.InvitationKey = invitationKeys[0]
	wr.LoginInfo.Invitation = &invitations[0]
	return nil
}

// Simple URL handler that prints out the invitation retrieved by
// LoginGetter, for testing.
func CheckLogin(c *gin.Context) {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	ctx := c.Request.Context()
	c.String(http.StatusOK, fmt.Sprintf("Invitation: %s", printInvitation(ctx, wr.LoginInfo.InvitationKey, wr.LoginInfo.Invitation)))
}

func handleLoginError(c *gin.Context) {
	wr, ok := c.MustGet("wrappedRequest").(*WrappedRequest)
	if !ok {
		log.Printf("could not get wrapped request from context")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	wr.Request.ParseForm()
	message_list, ok := wr.Request.Form["message"]
	var message string
	if ok {
		message = message_list[0]
	} else {
		message = "Login not found."
	}
	tpl := template.Must(template.New("").ParseFiles(
		"templates/main.html",
		"templates/bad_login.html"))
	url, _ := user.LoginURL(c.Request.Context(), "/")
	data := wr.MakeTemplateData(map[string]any{
		"Message":  message,
		"LoginURL": url,
	})
	if err := tpl.ExecuteTemplate(c.Writer, "bad_login.html", data); err != nil {
		log.Printf("%v", err)
	}
}

func handleLogout(c *gin.Context) {
	wr, ok := c.MustGet("wrappedRequest").(*WrappedRequest)
	if !ok {
		log.Printf("could not get wrapped request from context")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	wr.SetSessionValue("code", nil)
	wr.SetSessionValue("person", nil)
	wr.SaveSession()
	c.Redirect(http.StatusFound, "/")
}

func handleResendInvitation(c *gin.Context) {
	wr, ok := c.MustGet("wrappedRequest").(*WrappedRequest)
	if !ok {
		log.Printf("could not get wrapped request from context")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	dsclient := dsclient.FromContext(c.Request.Context())

	wr.Request.ParseForm()
	emailAddresses, ok := wr.Request.PostForm["emailAddress"]
	if !ok || len(emailAddresses) != 1 {
		c.Redirect(http.StatusFound, loginErrorPage+"?message=Bad form input.")
		return
	}
	q := datastore.NewQuery("Person").FilterField("Email", "=", emailAddresses[0])
	var people []person.Person
	_, err := dsclient.GetAll(c.Request.Context(), q, &people)
	if err != nil {
		log.Printf("%v", err)
		c.Redirect(http.StatusFound, loginErrorPage+"?message=Query error (contact admin: code RIGPER).")
	}
	// NOTE: This does not give an error message if the email
	// address is not found, so no one can probe the system for
	// people they know. This may be a bad UI, but it is good
	// privacy.
	if len(people) == 1 {
		loginUrl := MakeLoginUrl(&people[0], true)
		data := map[string]any{
			"Event":     *wr.Event,
			"LoginLink": loginUrl,
		}
		header := MailHeaderInfo{
			To:      EmailForPerson(&people[0]),
			BccSelf: false,
		}
		SendMail(c, "resendInvitation", data, header)
	}
	c.Redirect(http.StatusFound, resentInvitationPage+"?emailAddress="+emailAddresses[0])
}

func handleResentInvitation(c *gin.Context) {
	wr, ok := c.MustGet("wrappedRequest").(*WrappedRequest)
	if !ok {
		log.Printf("could not get wrapped request from context")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	wr.Request.ParseForm()
	emailAddresses, ok := wr.Request.Form["emailAddress"]
	if !ok || len(emailAddresses) != 1 {
		c.Redirect(http.StatusFound, loginErrorPage+"?message=An error occurred.")
		return
	}
	data := wr.MakeTemplateData(map[string]any{
		"ResentAddress": emailAddresses[0],
	})
	tpl := template.Must(template.New("").ParseFiles(
		"templates/main.html",
		"templates/resentInvitation.html"))
	if err := tpl.ExecuteTemplate(c.Writer, "resentInvitation.html", data); err != nil {
		log.Printf("%v", err)
	}
}

func MakeLoginUrl(p *person.Person, absolute bool) string {
	var prefix string
	if absolute {
		prefix = "http://psr.shabsin.com"
	}
	return prefix + "/login?loginCode=" + p.LoginCode
}