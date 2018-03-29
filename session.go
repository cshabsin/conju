package conju

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

// TODO(cshabsin): Figure out how to store the secret in the datastore
// instead of source.
var store = sessions.NewCookieStore([]byte("devmode_key_crsdms"))

type WrappedRequest struct {
	http.ResponseWriter
	*http.Request
	context.Context
	*sessions.Session
	hasRunEventGetter bool
	EventKey          *datastore.Key // TODO: stick these in EventInfo
	*Event
	*LoginInfo
}

type Getter func(*WrappedRequest) error

type Getters struct {
	Getters []Getter
}

type RedirectError struct {
	Target string
}

func (re RedirectError) Error() string {
	return fmt.Sprintf("Redirect to %s", re.Target)
}

func AddSessionHandler(url string, f func(WrappedRequest)) *Getters {
	getters := Getters{make([]Getter, 0)}
	http.HandleFunc(url, func(w http.ResponseWriter, r *http.Request) {
		sess, err := store.Get(r, "conju")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ctx := appengine.NewContext(r)
		wr := WrappedRequest{w, r, ctx, sess, false, nil, nil, nil}
		for _, getter := range getters.Getters {
			if err = getter(&wr); err != nil {
				if redirect, ok := err.(RedirectError); ok {
					http.Redirect(w, r, redirect.Target, http.StatusFound)
					return
				}
				// TODO: Probably not internal server error
				http.Error(w, err.Error(), http.StatusInternalServerError)
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
	w.Values[key] = value
}

// Call SaveSession before writing any output to writer.
func (w *WrappedRequest) SaveSession() error {
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
