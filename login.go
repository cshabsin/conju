package conju

import (
	"fmt"
	"html/template"
	"math/rand"
	"net/http"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
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

const loginErrorPage = "/loginError"
const resentInvitationPage = "/resentInvitation"

func handleLogin(urlTarget string) func(wr WrappedRequest) {
	return func(wr WrappedRequest) {
		handleLoginInner(wr, urlTarget)
	}
}

// When a user navigates to the login link and provides the given code
// string, the system validates the login code against the Person
// table, and either puts the login code into the session, or writes
// an error. On error, we display an error page with help. On success,
// we redirect to urlTarget.
func handleLoginInner(wr WrappedRequest, urlTarget string) {
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
	var people []Person
	peopleKeys, err := datastore.NewQuery("Person").Filter("LoginCode =", lc[0]).GetAll(wr.Context, &people)
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

// LoginGetter validates the login code from the session, looking up
// the Person with the matching code. Then it finds the Invitation in
// the current Event (per the WrappedRequest field Event) that
// references that person. It stores the Person and Invitation (key
// and value) into the WrappedRequest's LoginInfo.  This getter will
// redirect to the login screen if the LoginCode is not found in the
// database.
//
// If EventGetter has not been called, LoginGetter calls it.
func PersonGetter(wr *WrappedRequest) error {
	if !wr.hasRunEventGetter {
		if err := EventGetter(wr); err != nil {
			return err
		}
	}
	if wr.LoginInfo != nil {
		return nil // This has already been run.
	}
	code, ok := wr.Values["code"].(string)
	if !ok {
		return RedirectError{loginErrorPage +
			"?message=Please use the link from your email to log in."}
	}
	personKeyEncoded, ok := wr.Values["person"].(string)
	var person Person
	var personKey *datastore.Key
	if !ok {
		var people []Person
		peopleKeys, err := datastore.NewQuery("Person").Filter("LoginCode =", code).GetAll(wr.Context, &people)
		if err != nil {
			return err
		}
		if len(people) == 0 {
			return RedirectError{loginErrorPage +
				"?message=Person not found for loginCode."}
		} else if len(people) > 1 {
			return RedirectError{loginErrorPage +
				"?message=DB Error: loginCode collision."}
		}
		wr.SetSessionValue("person", peopleKeys[0].Encode())
		person = people[0]
		personKey = peopleKeys[0]
	} else {
		var err error
		personKey, err = datastore.DecodeKey(personKeyEncoded)
		err = datastore.Get(wr.Context, personKey, &person)
		if err != nil {
			return err
		}
		if person.LoginCode != code {
			return RedirectError{loginErrorPage +
				"?message=Something went out of sync. Please log in " +
				"again using the link from your email."}
		}
	}
	wr.LoginInfo = &LoginInfo{nil, nil, personKey, &person}
	return nil
}

func InvitationGetter(wr *WrappedRequest) error {
	if wr.LoginInfo == nil {
		if err := PersonGetter(wr); err != nil {
			return err
		}
	}
	var invitations []Invitation
	invitationKeys, err := datastore.NewQuery("Invitation").
		Filter("Invitees =", wr.LoginInfo.PersonKey).
		Filter("Event =", wr.EventKey).
		GetAll(wr.Context, &invitations)
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
func checkLogin(wr WrappedRequest) {
	wr.ResponseWriter.Write([]byte(fmt.Sprintf("Invitation: %s", printInvitation(wr.Context, *wr.LoginInfo.InvitationKey, *wr.LoginInfo.Invitation))))
}

func handleLoginError(wr WrappedRequest) {
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
	data := wr.MakeTemplateData(map[string]interface{}{
		"Message": message,
	})
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "bad_login.html", data); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}

func handleResendInvitation(wr WrappedRequest) {
	wr.Request.ParseForm()
	emailAddresses, ok := wr.Request.PostForm["emailAddress"]
	if !ok || len(emailAddresses) != 1 {
		http.Redirect(wr.ResponseWriter, wr.Request,
			loginErrorPage+"?message=Bad form input.", http.StatusFound)
		return
	}
	q := datastore.NewQuery("Person").Filter("Email =", emailAddresses[0])
	var people []Person
	_, err := q.GetAll(wr.Context, &people)
	if err != nil {
		log.Errorf(wr.Context, "%v", err)
		http.Redirect(wr.ResponseWriter, wr.Request,
			loginErrorPage+"?message=Query error (contact admin: code RIGPER).",
			http.StatusFound)
	}
	// NOTE: This does not give an error message if the email
	// address is not found, so no one can probe the system for
	// people they know. This may be a bad UI, but it is good
	// privacy.
	if len(people) == 1 {
		loginUrl := makeLoginUrl(&people[0])
		data := map[string]interface{}{
			"LoginLink": loginUrl,
		}
		header := MailHeaderInfo{
			To: []string{people[0].Email},
		}
		sendMail(wr.Context, "resendInvitation", data, nil, header)
	}
	// TODO: Make a resentInvitation.html template explaining that
	// if they don't get email in a minute or two from us, they
	// should contact us to find out which email addresses of
	// theirs we have on file.
	http.Redirect(wr.ResponseWriter, wr.Request,
		resentInvitationPage+"?emailAddress="+emailAddresses[0],
		http.StatusFound)
}

func handleResentInvitation(wr WrappedRequest) {
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
		log.Errorf(wr.Context, "%v", err)
	}
}

func makeLoginUrl(p *Person) string {
	return SiteLink + "/login?loginCode=" + p.LoginCode
}
