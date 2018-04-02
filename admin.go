package conju

import (
	"errors"
	"fmt"

	"google.golang.org/appengine/user"
)

type AdminInfo struct {
	*user.User
}

func AdminGetter(wr *WrappedRequest) error {
	u := user.Current(wr.Context)
	if u == nil {
		url, _ := user.LoginURL(wr.Context, wr.URL.RequestURI())
		fmt.Fprintf(wr.ResponseWriter, `<a href="%s">Sign in or register</a>`, url)
		return errors.New("Not Admin.")
	}
	if u.Admin { // u.Email == "cshabsin@gmail.com" || u.Email == "dana.m.scott@gmail.com" {
		wr.AdminInfo = &AdminInfo{u}
		return nil
	}
	logout_url, err := user.LogoutURL(wr.Context, wr.URL.RequestURI())
	if err != nil {
		return err
	}
	return errors.New(fmt.Sprintf(
		`User %s is not an authorized administrator. <a href="%s">Click to sign out</a>.`,
		u.Email, logout_url))
}
