package conju

import (
	"net/http"

	"github.com/gorilla/sessions"
)

// TODO(cshabsin): Figure out how to store the secret in the datastore
// instead of source.
var store = sessions.NewCookieStore([]byte("devmode_key_crsdms"))

type WrappedRequest struct {
	http.ResponseWriter
	*http.Request
	*sessions.Session
}

func AddSessionHandler(url string, f func(WrappedRequest)) {
	http.HandleFunc(url, func(w http.ResponseWriter, r *http.Request) {
		sess, err := store.Get(r, "conju")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		f(WrappedRequest{w, r, sess})
		sess.Save(r, w)
	})
}
