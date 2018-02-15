package conju

import (
	"context"
	"errors"
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
	cachedEvent *Event
}

func AddSessionHandler(url string, f func(WrappedRequest)) {
	http.HandleFunc(url, func(w http.ResponseWriter, r *http.Request) {
		sess, err := store.Get(r, "conju")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ctx := appengine.NewContext(r)
		if sess.Values["event"] == nil {
			k, err := CurrentEventKey(ctx)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			sess.Values["event"] = k.Encode()
		}
		f(WrappedRequest{w, r, ctx, sess, nil})
	})
}

func (w *WrappedRequest) SaveSession() error {
	return w.Session.Save(w.Request, w.ResponseWriter)
}

func (w *WrappedRequest) CurrentEvent() (*Event, error) {
	if w.cachedEvent != nil {
		return w.cachedEvent, nil
	}
	encoded_key, ok := w.Values["event"].(string)
	if !ok {
		return nil, errors.New("Event not found in session.")
	}
	key, err := datastore.DecodeKey(encoded_key)
	if err != nil {
		return nil, err
	}
	var e Event
	err = datastore.Get(w.Context, key, &e)
	if err != nil {
		return nil, err
	}
	w.cachedEvent = &e
	return &e, nil
}

