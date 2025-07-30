package conju

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/datastore"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/sendgrid/sendgrid-go"
	"google.golang.org/appengine/v2/user"

	"github.com/cshabsin/conju/conju/dsclient"
	"github.com/cshabsin/conju/conju/secretmanager"
	"github.com/cshabsin/conju/model/event"
)

var store *sessions.CookieStore

func handleWarmup(ctx context.Context, wr *WrappedRequest) {
	// TODO - don't run this through sessionizer, so we get to do the initialization instead of having it done with the lazy code there.
	if store != nil {
		return
	}
	if err := initializeCookieStore(ctx); err != nil {
		log.Printf("Could not initialize secret manager in warmup: %v", err)
		http.Error(wr.ResponseWriter, err.Error(), http.StatusInternalServerError)
		store = sessions.NewCookieStore([]byte("devmode_key_crsdms"))
	}
}

func initializeCookieStore(ctx context.Context) error {
	log.Printf("initializing cookie store")
	smClient, err := secretmanager.FromContext(ctx)
	if err != nil {
		return fmt.Errorf("Error getting secret manager: %w", err)
	}
	secrets, err := smClient.GetVersionsWithinGracePeriod(ctx, "gorilla_cookie_auth_key", 0)
	if err != nil {
		return fmt.Errorf("Error getting gorilla cookie auth key: %w", err)
	}
	var secretArgs [][]byte
	for _, secret := range secrets {
		// NewCookieStore takes arg pairs - an auth key and an encryption key.
		// Right now we're not doing encryption keys, for simplicity.
		secretArgs = append(secretArgs, secret)
		secretArgs = append(secretArgs, nil)
	}
	store = sessions.NewCookieStore(secretArgs...)
	return nil
}

type WrappedRequest struct {
	*gin.Context

	EmailClient *sendgrid.Client

	*sessions.Session
	hasRunEventGetter bool
	EventKey          *datastore.Key // TODO: stick these in EventInfo
	*event.Event
	*user.User
	*LoginInfo
	TemplateData  map[string]any
	SenderAddress *string
	BccAddress    *string
	ErrorAddress  *string
	*BookingInfo
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



// TODO(cshabsin): Add check for whether the wrapped request has
// already written the header (in which case emit a warning or
// something since the change to the value will not be saved.
func (w *WrappedRequest) SetSessionValue(key string, value any) {
	w.Session.Values[key] = value
}

// Call SaveSession before writing any output to writer.
func (w *WrappedRequest) SaveSession() error {
	return w.Session.Save(w.Request, w.Writer)
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

func (w WrappedRequest) MakeTemplateData(extraVals map[string]any) map[string]any {
	vals := map[string]any{}
	for k, v := range w.TemplateData {
		vals[k] = v
	}
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


type BookingInfo struct {
	// map of booking key ID to booking object
	BookingKeyMap map[int64]Booking

	// map of person ID to booking ID
	PersonToBookingMap map[int64]int64
}

func (wr *WrappedRequest) GetBookingInfo(ctx context.Context) *BookingInfo {
	if wr.BookingInfo != nil {
		return wr.BookingInfo
	}
	wr.BookingInfo = GetBookingInfo(ctx, wr.Event)
	return wr.BookingInfo
}

func GetBookingInfo(ctx context.Context, ev *event.Event) *BookingInfo {
	client := dsclient.FromContext(ctx)
	if client == nil {
		log.Println("GetBookingInfo called with nil client")
		return nil
	}
	// Load all bookings for the event.
	var bookings []Booking
	q := datastore.NewQuery("Booking").Ancestor(ev.Key)
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
	return &BookingInfo{bookingKeyToBookingMap, personToBookingMap}
}
