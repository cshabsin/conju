package conju

import (
	"context"
	"fmt"

	"google.golang.org/appengine/datastore"
)

// This file defines a set of EmailDistributors, which the
// handleSendMail function uses to

type EmailSender func(context.Context, map[string]interface{}, MailHeaderInfo) error

type EmailDistributor func(WrappedRequest, EmailSender) error
type EmailDistributorEntry struct {
	NeedsConfirm bool
	Distribute   EmailDistributor
}

var AllDistributors = map[string]EmailDistributorEntry{
	"SelfOnly":          {false, SelfOnlyDistributor},
	"AllInviteesDryRun": {false, AllInviteesDryRunDistributor},
	"AllInvitees*REAL*": {true, AllInviteesDistributor},
}

func SelfOnlyDistributor(wr WrappedRequest, sender EmailSender) error {
	realizedInvitation := makeRealizedInvitation(wr.Context, *wr.LoginInfo.InvitationKey,
		*wr.LoginInfo.Invitation)
	emailData := map[string]interface{}{
		"Event":      wr.Event,
		"Invitation": realizedInvitation,
		"Person":     wr.LoginInfo.Person,
	}
	err := sender(wr.Context, emailData, MailHeaderInfo{To: []string{wr.LoginInfo.Person.Email}})
	return err
}

func AllInviteesDryRunDistributor(wr WrappedRequest, sender EmailSender) error {
	wr.ResponseWriter.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(wr.ResponseWriter, "Looking up all invitees...<br>")

	q := datastore.NewQuery("Invitation").Filter("Event =", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := q.GetAll(wr.Context, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(wr.Context, *invitationKeys[i],
			*invitations[i])
		for _, p := range realizedInvitation.Invitees {
			if p.Person.Email == "" {
				continue
			}
			emailData := map[string]interface{}{
				"Event":      wr.Event,
				"Invitation": realizedInvitation,
				"Person":     &p.Person,
			}
			fmt.Fprintf(wr.ResponseWriter, "Sending email for %s to %s.<br>", p.Person.Email, wr.LoginInfo.Person.Email)
			err := sender(wr.Context, emailData, MailHeaderInfo{To: []string{wr.LoginInfo.Person.Email}})
			if err != nil {
				fmt.Fprintf(wr.ResponseWriter, "Error emailing %s: %v", p.Person.Email, err)
				return err
			}
		}
	}
	return nil
}

func AllInviteesDistributor(wr WrappedRequest, sender EmailSender) error {
	wr.ResponseWriter.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(wr.ResponseWriter, "Looking up all invitees...<br>")

	q := datastore.NewQuery("Invitation").Filter("Event =", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := q.GetAll(wr.Context, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(wr.Context, *invitationKeys[i],
			*invitations[i])
		for _, p := range realizedInvitation.Invitees {
			if p.Person.Email == "" {
				continue
			}
			emailData := map[string]interface{}{
				"Event":      wr.Event,
				"Invitation": realizedInvitation,
				"Person":     &p.Person,
			}
			fmt.Fprintf(wr.ResponseWriter, "Sending email for %s.<br>", p.Person.Email)
			err := sender(wr.Context, emailData, MailHeaderInfo{To: []string{p.Person.Email}})
			if err != nil {
				fmt.Fprintf(wr.ResponseWriter, "Error emailing %s: %v", p.Person.Email, err)
				return err
			}
		}
	}
	return nil
}
