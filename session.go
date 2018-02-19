package conju

import (
	"context"
	"net/http"

	"github.com/gorilla/sessions"
	"google.golang.org/appengine"
)

// TODO(cshabsin): Figure out how to store the secret in the datastore
// instead of source.
var store = sessions.NewCookieStore([]byte("devmode_key_crsdms"))

type WrappedRequest struct {
	http.ResponseWriter
	*http.Request
	context.Context
	*sessions.Session
	*Event
}

type Getter func(*WrappedRequest) error

type Getters struct {
	Getters []Getter
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
		wr := WrappedRequest{w, r, ctx, sess, nil}
		for _, getter := range(getters.Getters) {
			if err = getter(&wr); err != nil {
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
