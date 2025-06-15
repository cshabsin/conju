package conju

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/datastore"
	"github.com/gorilla/sessions"
	"github.com/sendgrid/sendgrid-go"
	"google.golang.org/appengine/user"

	"github.com/cshabsin/conju/conju/dsclient"
	"github.com/cshabsin/conju/model/event"
)

// TODO(cshabsin): Figure out how to store the secret in the datastore
// instead of source.
var store = sessions.NewCookieStore([]byte("devmode_key_crsdms"))

type WrappedRequest struct {
	EmailClient     *sendgrid.Client
	DatastoreClient *datastore.Client

	ResponseWriter WrappedResponseWriter
	*http.Request
	*sessions.Session
	hasRunEventGetter bool
	EventKey          *datastore.Key // TODO: stick these in EventInfo
	*event.Event
	*user.User
	*LoginInfo
	TemplateData  map[string]interface{}
	SenderAddress *string
	BccAddress    *string
	ErrorAddress  *string
	*BookingInfo
}

type Getter func(context.Context, *WrappedRequest) error

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
	return "Done processing, do not continue."
}

type Sessionizer struct {
	Client *datastore.Client
}

func (s Sessionizer) AddSessionHandler(url string, f func(context.Context, WrappedRequest)) *Getters {
	var getters Getters
	getters.Getters = []Getter{EventGetter}
	http.HandleFunc(url, func(w http.ResponseWriter, r *http.Request) {
		ctx := dsclient.WrapContext(r.Context(), s.Client)
		log.Printf("Handling request %v", r.URL.Path)
		wrw := NewWrappedResponseWriter(w)
		sess, err := store.Get(r, "conju")
		if err != nil {
			log.Printf("Could not get session from store: %v", err)
			// TODO: Clear session instead of erroring out?
			http.Error(wrw, err.Error(), http.StatusInternalServerError)
			return
		}
		u := user.Current(ctx)
		wr := WrappedRequest{
			ResponseWriter: wrw,
			Request:        r,
			Session:        sess,
			User:           u,
			TemplateData: map[string]interface{}{
				"User": u,
			},
			DatastoreClient: s.Client,
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
			if err = getter(ctx, &wr); err != nil {
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
				log.Printf("Getter (index %d) returned an error on request %s: %v",
					i, wr.Request.URL.Path, err)
				// TODO: Probably not internal server error
				http.Error(wrw, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		f(ctx, wr)
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
		log.Printf("SetSessionValue called after header written. key %s, value %v", key, value)
	}
	w.Session.Values[key] = value
}

// Call SaveSession before writing any output to writer.
func (w *WrappedRequest) SaveSession() error {
	if w.ResponseWriter.HasWrittenHeader() {
		log.Printf("SaveSession called after header written.")
	}
	return w.Session.Save(w.Request, w.ResponseWriter)
}

// Attempts to read a datastore key from the request session, returning:
//   - a key value (if the value is present and valid)
//   - nil (if the value is not present)
//   - nil and an error (if the value is invalid)
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

func (w *WrappedRequest) GetEmailClient() *sendgrid.Client {
	if w.EmailClient == nil {
		w.EmailClient = sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
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

func (w WrappedRequest) GetEnvForTemplates() map[string]string {
	rc := make(map[string]string)
	for _, s := range []string{"GOOGLE_WALLET_ADDRESS", "VENMO_ADDRESS", "PAYPAL_ADDRESS", "PAYPAL_URL", "DISCORD_URL"} {
		rc[s] = os.Getenv(s)
	}
	return rc
}

func (w WrappedRequest) GetHost() string {
	w.Request.ParseForm()
	host, ok := w.Request.Form["host_override"]
	if ok {
		return host[0]
	}
	// TODO: add debug override.
	host, ok = w.Header["Host"]
	if !ok || len(host) == 0 {
		return ""
	}
	return strings.ToLower(host[0])
}

// / WrappedResponseWriter simply records when the header has been
// / written, so SetSessionValue can check and error when this has
// / occurred.
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

type BookingInfo struct {
	// map of booking key ID to booking object
	BookingKeyMap map[int64]Booking

	// map of person ID to booking ID
	PersonToBookingMap map[int64]int64
}

func (wr *WrappedRequest) GetBookingInfo(ctx context.Context) *BookingInfo {
	client := dsclient.FromContext(ctx)
	if client == nil {
		log.Println("GetBookingInfo called with nil client")
		return nil
	}
	if wr.BookingInfo != nil {
		return wr.BookingInfo
	}
	// Load all bookings for the event.
	var bookings []Booking
	q := datastore.NewQuery("Booking").Ancestor(wr.EventKey)
	allBookingKeys, err := client.GetAll(ctx, q, &bookings)
	if err != nil {
		log.Printf("Error reading all booking keys: %v", err)
		return nil
	}

	// Construct lookup maps on bookings - booking key to booking, person to booking.
	bookingKeyToBookingMap := make(map[int64]Booking)
	personToBookingMap := make(map[int64]int64)
	for b, booking := range bookings {
		bookingKeyToBookingMap[allBookingKeys[b].ID] = booking
		for _, person := range booking.Roommates {
			personToBookingMap[person.ID] = allBookingKeys[b].ID
		}
	}
	wr.BookingInfo = &BookingInfo{bookingKeyToBookingMap, personToBookingMap}
	return wr.BookingInfo
}
