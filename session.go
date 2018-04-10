package conju

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"gopkg.in/sendgrid/sendgrid-go.v2"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"google.golang.org/appengine/user"
)

// TODO(cshabsin): Figure out how to store the secret in the datastore
// instead of source.
var store = sessions.NewCookieStore([]byte("devmode_key_crsdms"))

type WrappedRequest struct {
	ResponseWriter WrappedResponseWriter
	*http.Request
	context.Context
	*sessions.Session
	hasRunEventGetter bool
	EventKey          *datastore.Key // TODO: stick these in EventInfo
	*Event
	*user.User
	*LoginInfo
	TemplateData  map[string]interface{}
	SenderAddress *string
	BccAddress    *string
	ErrorAddress  *string
	EmailClient   *sendgrid.SGClient
}

type Getter func(*WrappedRequest) error

type Getters struct {
	Getters []Getter
}

// Getters should return this error to generate a HTTP redirect.
type RedirectError struct {
	Target string
}

func (re RedirectError) Error() string {
	return fmt.Sprintf("Redirect to %s", re.Target)
}

// Getters should return this error to indicate an error has occurred
// that has been reported cleanly.
type DoneProcessingError struct {
}

func (dpe DoneProcessingError) Error() string {
	return fmt.Sprintf("Done processing, do not continue.")
}

func AddSessionHandler(url string, f func(WrappedRequest)) *Getters {
	var getters Getters
	getters.Getters = []Getter{EventGetter}
	http.HandleFunc(url, func(w http.ResponseWriter, r *http.Request) {
		wrw := NewWrappedResponseWriter(w)
		sess, err := store.Get(r, "conju")
		if err != nil {
			// TODO: Clear session instead of erroring out?
			http.Error(wrw, err.Error(), http.StatusInternalServerError)
			return
		}
		ctx := appengine.NewContext(r)
		u := user.Current(ctx)
		wr := WrappedRequest{
			ResponseWriter: wrw,
			Request:        r,
			Context:        ctx,
			Session:        sess,
			User:           u,
			TemplateData: map[string]interface{}{
				"User": u,
			},
		}
		if u != nil {
			logoutUrl, err := user.LogoutURL(ctx, wr.URL.RequestURI())
			if err == nil {
				wr.TemplateData["LogoutLink"] = logoutUrl
			}
		}
		wr.TemplateData["IsAdminUser"] = wr.IsAdminUser()
		// TODO: make this always true once we go live.
		wr.TemplateData["ShowRsvp"] = wr.IsAdminUser()
		for i, getter := range getters.Getters {
			if err = getter(&wr); err != nil {
				if redirect, ok := err.(RedirectError); ok {
					http.Redirect(wrw, r, redirect.Target, http.StatusFound)
					return
				}
				if _, ok := err.(DoneProcessingError); ok {
					return
				}
				sendErrorMail(wr, fmt.Sprintf(
					"Getter (index %d) returned an error on request %s: %v",
					i, wr.Request.URL.Path, err))
				// TODO: Probably not internal server error
				http.Error(wrw, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		f(wr)
	})
	return &getters
}

func (g *Getters) Needs(getter Getter) *Getters {
	g.Getters = append(g.Getters, getter)
	return g
}

// TODO(cshabsin): Add check for whether the wrapped request has
// already written the header (in which case emit a warning or
// something since the change to the value will not be saved.
func (w *WrappedRequest) SetSessionValue(key string, value interface{}) {
	if w.ResponseWriter.HasWrittenHeader() {
		log.Errorf(w.Context, "SetSessionValue called after header written. key %s, value %v", key, value)
	}
	w.Session.Values[key] = value
}

// Call SaveSession before writing any output to writer.
func (w *WrappedRequest) SaveSession() error {
	if w.ResponseWriter.HasWrittenHeader() {
		log.Errorf(w.Context, "SaveSession called after header written.")
	}
	return w.Session.Save(w.Request, w.ResponseWriter)
}

// Attempts to read a datastore key from the request session, returning:
//  - a key value (if the value is present and valid)
//  - nil (if the value is not present)
//  - nil and an error (if the value is invalid)
func (w *WrappedRequest) RetrieveKeyFromSession(values_field string) (*datastore.Key, error) {
	encoded_key_interface := w.Values[values_field]
	if encoded_key_interface == nil {
		return nil, nil
	}
	encoded_key, ok := encoded_key_interface.(string)
	if !ok {
		return nil, nil
	}
	return datastore.DecodeKey(encoded_key)

}

func (w WrappedRequest) IsAdminUser() bool {
	if w.User == nil {
		return false
	}
	return w.User.Admin
}

func (w WrappedRequest) MakeTemplateData(extraVals map[string]interface{}) map[string]interface{} {
	vals := w.TemplateData
	for k, v := range extraVals {
		vals[k] = v
	}
	return vals
}

func (w *WrappedRequest) GetEmailClient() *sendgrid.SGClient {
	if w.EmailClient == nil {
		w.EmailClient = sendgrid.NewSendGridClientWithApiKey(os.Getenv("SENDGRID_API_KEY"))
		w.EmailClient.Client = urlfetch.Client(w.Context)
	}
	return w.EmailClient
}

// Also receives the rsvp change status.
func (w WrappedRequest) GetSenderAddress() string {
	return os.Getenv("SENDER_ADDRESS")
}

func (w WrappedRequest) GetBccAddress() string {
	return os.Getenv("BCC_ADDRESS")
}

func (w WrappedRequest) GetErrorAddress() string {
	return os.Getenv("ERROR_ADDRESS")
}

/// WrappedResponseWriter simply records when the header has been
/// written, so SetSessionValue can check and error when this has
/// occurred.
type WrappedResponseWriter struct {
	http.ResponseWriter
	stats *responseWriterStats
}

type responseWriterStats struct {
	hasWrittenHeader bool
}

func NewWrappedResponseWriter(w http.ResponseWriter) WrappedResponseWriter {
	return WrappedResponseWriter{w, &responseWriterStats{false}}
}

func (wrw WrappedResponseWriter) Header() http.Header {
	return wrw.ResponseWriter.Header()
}

func (wrw WrappedResponseWriter) Write(b []byte) (int, error) {
	wrw.stats.hasWrittenHeader = true
	return wrw.ResponseWriter.Write(b)
}

func (wrw WrappedResponseWriter) WriteHeader(statuscode int) {
	wrw.stats.hasWrittenHeader = true
	wrw.ResponseWriter.WriteHeader(statuscode)
}

func (wrw WrappedResponseWriter) HasWrittenHeader() bool {
	return wrw.stats.hasWrittenHeader
}
