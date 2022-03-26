package conju

import (
	"context"
	"fmt"

	"google.golang.org/appengine/user"
)

// TODO(cshabsin): Replace error pages with templates.
func AdminGetter(ctx context.Context, wr *WrappedRequest) error {
	if wr.IsAdminUser() {
		return nil
	}
	u := wr.User
	if u == nil {
		url, _ := user.LoginURL(ctx, wr.URL.RequestURI())
		wr.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(wr.ResponseWriter, `This page requires administrator access. Please <a href="%s">Sign in</a>.`, url)
		return DoneProcessingError{}
	}
	logout_url, err := user.LogoutURL(ctx, wr.URL.RequestURI())
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
