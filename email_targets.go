package conju

import (
	"context"
	"fmt"
)

// This file defines a set of EmailDistributors, which the
// handleSendMail function uses to

type EmailSender func(context.Context, map[string]interface{}, MailHeaderInfo) error

type EmailDistributor func(WrappedRequest, EmailSender) error

var AllDistributors = map[string]EmailDistributor{
	"SelfOnly":          SelfOnlyDistributor,
	"AllInviteesDryRun": AllInviteesDryRunDistributor,
	"AllInvitees*REAL*": AllInviteesDistributor,
}

func SelfOnlyDistributor(wr WrappedRequest, sender EmailSender) error {
	realizedInvitation := makeRealizedInvitation(wr.Context, *wr.LoginInfo.InvitationKey,
		*wr.LoginInfo.Invitation)
	emailData := map[string]interface{}{
		"Event":      wr.Event,
		"Invitation": realizedInvitation,
		"Person":     wr.LoginInfo.Person,
		"LoginLink":  makeLoginUrl(wr.LoginInfo.Person),
	}
	err := sender(wr.Context, emailData, MailHeaderInfo{To: []string{wr.LoginInfo.Person.Email}})
	return err
}

func AllInviteesDryRunDistributor(wr WrappedRequest, sender EmailSender) error {
	wr.ResponseWriter.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(wr.ResponseWriter, "Looking up all invitees...<br>")
	// TODO: for all invitees in wr, call sender on the data built up from
	// that invitee, but with "dry run" values for recipients.
	return nil
}

func AllInviteesDistributor(wr WrappedRequest, sender EmailSender) error {
	// TODO: for all invitees in wr, call sender on the data built up
	// from that invitee.
	return nil
}
