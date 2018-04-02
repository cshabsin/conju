package conju

import (
	"fmt"

	"google.golang.org/appengine/user"
)

type AdminInfo struct {
	*user.User
}

// TODO(cshabsin): Replace error pages with templates.
func AdminGetter(wr *WrappedRequest) error {
	u := user.Current(wr.Context)
	if u == nil {
		url, _ := user.LoginURL(wr.Context, wr.URL.RequestURI())
		wr.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(wr.ResponseWriter, `This page requires administrator access. Please <a href="%s">Sign in</a>.`, url)
		return DoneProcessingError{}
	}
	if u.Admin {
		wr.AdminInfo = &AdminInfo{u}
		return nil
	}
	logout_url, err := user.LogoutURL(wr.Context, wr.URL.RequestURI())
	if err != nil {
		return err
	}
	wr.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(
		wr.ResponseWriter,
		`This page requires administrator access.<br>User <code>&lt;%s&gt;</code> is not an authorized administrator.<p>Please <a href="%s">sign out</a> to try another account.`,
		u.Email, logout_url)
	return DoneProcessingError{}
}
