package conju

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/cshabsin/conju/model/person"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/user"
)

type LoginInfo struct {
	InvitationKey *datastore.Key
	*Invitation
	PersonKey *datastore.Key
	*person.Person
}

const loginErrorPage = "/loginError"
const resentInvitationPage = "/resentInvitation"

func handleLogin(urlTarget string) func(ctx context.Context, wr WrappedRequest) {
	return func(ctx context.Context, wr WrappedRequest) {
		handleLoginInner(ctx, wr, urlTarget)
	}
}

// When a user navigates to the login link and provides the given code
// string, the system validates the login code against the Person
// table, and either puts the login code into the session, or writes
// an error. On error, we display an error page with help. On success,
// we redirect to urlTarget.
func handleLoginInner(ctx context.Context, wr WrappedRequest, urlTarget string) {
	// TODO(cshabsin): Read "message" CGI arg if present and
	// display it. Prettify this page in general, using templates.
	url_q := wr.URL.Query()
	lc, ok := url_q["loginCode"]
	if !ok {
		http.Redirect(wr.ResponseWriter, wr.Request, loginErrorPage+
			"?message=Login is required for this section of our site.  Please use the link from your email to log in.",
			http.StatusFound)
		return
	}
	var people []person.Person
	peopleKeys, err := datastore.NewQuery("Person").Filter("LoginCode =", lc[0]).GetAll(ctx, &people)
	if err != nil {
		http.Redirect(wr.ResponseWriter, wr.Request,
			fmt.Sprintf("%s?message=DB error looking you up: %v", loginErrorPage, err),
			http.StatusFound)
		return
	}
	if len(peopleKeys) == 0 {
		http.Redirect(wr.ResponseWriter, wr.Request, loginErrorPage+
			"?message=Login not recognized.",
			http.StatusFound)
		return
	}
	if len(peopleKeys) > 1 {
		http.Redirect(wr.ResponseWriter, wr.Request, loginErrorPage+
			"?message=DB Error: multiple invitations found.",
			http.StatusFound)
	}
	wr.SetSessionValue("code", lc[0])
	wr.SetSessionValue("person", peopleKeys[0].Encode())
	wr.SaveSession()
	http.Redirect(wr.ResponseWriter, wr.Request, urlTarget, http.StatusFound)
}

func getPersonFromEncodedKey(ctx context.Context, wr *WrappedRequest) (*datastore.Key, *person.Person, error) {
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
	err = datastore.Get(ctx, personKey, &pers)
	if err != nil {
		log.Printf("person get error: %v", err)
		return nil, nil, err
	}
	return personKey, &pers, nil
}

func getPersonFromLoggedInUser(ctx context.Context, wr *WrappedRequest) (*datastore.Key, *person.Person, error) {
	log.Printf("getPersonFromLoggedInUser")
	if wr.User == nil {
		log.Printf("not logged in")
		return nil, nil, errors.New("not logged in")
	}
	var people []*person.Person
	peopleKeys, err := datastore.NewQuery("Person").Filter("Email =", wr.User.Email).GetAll(ctx, &people)
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
	log.Printf("getPersonFromInvitationCode")
	code, ok := wr.Values["code"].(string)
	if !ok {
		log.Printf("invitation code not set")
		return nil, nil, errors.New("invitation code not set")
	}
	var people []*person.Person
	peopleKeys, err := datastore.NewQuery("Person").Filter("LoginCode =", code).GetAll(ctx, &people)
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
		return RedirectError{loginErrorPage + "?message=Please use the link from your invitation email to log in."}
	}
	var invitations []Invitation
	invitationKeys, err := datastore.NewQuery("Invitation").
		Filter("Invitees =", wr.LoginInfo.PersonKey).
		Filter("Event =", wr.EventKey).
		GetAll(ctx, &invitations)
	if err != nil {
		return err
	}
	if len(invitations) == 0 {
		return RedirectError{loginErrorPage + "?message=No invitation found for currently selected event"}
	} else if len(invitations) > 1 {
		return RedirectError{loginErrorPage + "?message=DB Error: multiple invitations found."}
	}

	wr.LoginInfo.InvitationKey = invitationKeys[0]
	wr.LoginInfo.Invitation = &invitations[0]
	return nil
}

// Simple URL handler that prints out the invitation retrieved by
// LoginGetter, for testing.
func CheckLogin(ctx context.Context, wr WrappedRequest) {
	wr.ResponseWriter.Write([]byte(fmt.Sprintf("Invitation: %s", printInvitation(ctx, wr.LoginInfo.InvitationKey, wr.LoginInfo.Invitation))))
}

func handleLoginError(ctx context.Context, wr WrappedRequest) {
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
	url, _ := user.LoginURL(ctx, "/")
	data := wr.MakeTemplateData(map[string]interface{}{
		"Message":  message,
		"LoginURL": url,
	})
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "bad_login.html", data); err != nil {
		log.Printf("%v", err)
	}
}

func handleLogout(ctx context.Context, wr WrappedRequest) {
	wr.SetSessionValue("code", nil)
	wr.SetSessionValue("person", nil)
	wr.SaveSession()
	http.Redirect(wr.ResponseWriter, wr.Request, "/", http.StatusFound)
}

func handleResendInvitation(ctx context.Context, wr WrappedRequest) {
	wr.Request.ParseForm()
	emailAddresses, ok := wr.Request.PostForm["emailAddress"]
	if !ok || len(emailAddresses) != 1 {
		http.Redirect(wr.ResponseWriter, wr.Request,
			loginErrorPage+"?message=Bad form input.", http.StatusFound)
		return
	}
	q := datastore.NewQuery("Person").Filter("Email =", emailAddresses[0])
	var people []person.Person
	_, err := q.GetAll(ctx, &people)
	if err != nil {
		log.Printf("%v", err)
		http.Redirect(wr.ResponseWriter, wr.Request,
			loginErrorPage+"?message=Query error (contact admin: code RIGPER).",
			http.StatusFound)
	}
	// NOTE: This does not give an error message if the email
	// address is not found, so no one can probe the system for
	// people they know. This may be a bad UI, but it is good
	// privacy.
	if len(people) == 1 {
		loginUrl := makeLoginUrl(&people[0], true)
		data := map[string]interface{}{
			"Event":     *wr.Event,
			"LoginLink": loginUrl,
		}
		header := MailHeaderInfo{
			To:      []string{people[0].Email},
			BccSelf: false,
		}
		sendMail(wr, "resendInvitation", data, header)
	}
	// TODO: Make a resentInvitation.html template explaining that
	// if they don't get email in a minute or two from us, they
	// should contact us to find out which email addresses of
	// theirs we have on file.
	http.Redirect(wr.ResponseWriter, wr.Request,
		resentInvitationPage+"?emailAddress="+emailAddresses[0],
		http.StatusFound)
}

func handleResentInvitation(ctx context.Context, wr WrappedRequest) {
	wr.Request.ParseForm()
	emailAddresses, ok := wr.Request.Form["emailAddress"]
	if !ok || len(emailAddresses) != 1 {
		http.Redirect(wr.ResponseWriter, wr.Request,
			loginErrorPage+"?message=An error occurred.", http.StatusFound)
		return
	}
	data := wr.MakeTemplateData(map[string]interface{}{
		"ResentAddress": emailAddresses[0],
	})
	tpl := template.Must(template.New("").ParseFiles(
		"templates/main.html",
		"templates/resentInvitation.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "resentInvitation.html", data); err != nil {
		log.Printf("%v", err)
	}
}

func makeLoginUrl(p *person.Person, absolute bool) string {
	var prefix string
	if absolute {
		prefix = "http://psr.shabsin.com"
	}
	return prefix + "/login?loginCode=" + p.LoginCode
}
