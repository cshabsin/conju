package conju

import (
	"fmt"

	"google.golang.org/appengine/log"
)

func handleLogin(wr WrappedRequest) {
	q := wr.URL.Query()
	lc, ok := q["loginCode"]
	if !ok {
		wr.ResponseWriter.Write([]byte("Please use the link from your email to log in."))
		return
	}
	wr.SetSessionValue("code", lc[0])
	log.Infof(wr.Context, "Set session value: %s -> %v", "code", lc[0])
	wr.SaveSession()
	wr.ResponseWriter.Write([]byte(fmt.Sprintf("Got loginCode: %s\n", lc[0])))
}

func checkLogin(wr WrappedRequest) {
	wr.ResponseWriter.Write([]byte(fmt.Sprintf("Invitation: %s", printInvitation(wr.Context, *wr.LoginInfo.InvitationKey, *wr.LoginInfo.Invitation))))
}
