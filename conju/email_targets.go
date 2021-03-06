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
	"AttendeesList":     {false, AttendeesListDistributor},
	"AttendeesDryRun":   {false, AttendeesDryRunDistributor},
	"Attendees*REAL*":   {true, AttendeesDistributor},
}

func SelfOnlyDistributor(wr WrappedRequest, sender EmailSender) error {
	realizedInvitation := makeRealizedInvitation(wr.Context, wr.LoginInfo.InvitationKey,
		wr.LoginInfo.Invitation)
	roomingInfo := getRoomingInfoWithInvitation(wr, wr.LoginInfo.Invitation, wr.LoginInfo.InvitationKey)
	fmt.Fprintf(wr.ResponseWriter, "Sending only to &lt;%s&gt;.<br>", wr.LoginInfo.Person.Email)
	emailData := map[string]interface{}{
		"Event":       wr.Event,
		"Invitation":  realizedInvitation,
		"Person":      wr.LoginInfo.Person,
		"RoomingInfo": roomingInfo,
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
		realizedInvitation := makeRealizedInvitation(wr.Context, invitationKeys[i],
			invitations[i])
		roomingInfo := getRoomingInfoWithInvitation(wr, invitations[i], invitationKeys[i])
		for _, p := range realizedInvitation.Invitees {
			if p.Person.Email == "" {
				continue
			}
			emailData := map[string]interface{}{
				"Event":       wr.Event,
				"Invitation":  realizedInvitation,
				"Person":      &p.Person,
				"RoomingInfo": roomingInfo,
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
		realizedInvitation := makeRealizedInvitation(wr.Context, invitationKeys[i],
			invitations[i])
		roomingInfo := getRoomingInfoWithInvitation(wr, invitations[i], invitationKeys[i])
		for _, p := range realizedInvitation.Invitees {
			if p.Person.Email == "" {
				continue
			}
			emailData := map[string]interface{}{
				"Event":       wr.Event,
				"Invitation":  realizedInvitation,
				"Person":      &p.Person,
				"RoomingInfo": roomingInfo,
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

func AttendeesListDistributor(wr WrappedRequest, sender EmailSender) error {
	wr.ResponseWriter.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(wr.ResponseWriter, "Looking up all attendees...<br>")

	q := datastore.NewQuery("Invitation").Filter("Event =", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := q.GetAll(wr.Context, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(wr.Context, invitationKeys[i],
			invitations[i])
		roomingInfo := getRoomingInfoWithInvitation(wr, invitations[i], invitationKeys[i])
		if roomingInfo == nil {
			continue
		}
		for _, p := range realizedInvitation.Invitees {
			if p.Person.Email == "" {
				continue
			}
			if _, found := roomingInfo.Attendees[p.Person.DatastoreKey.IntID()]; !found {
				continue
			}
			fmt.Fprintf(wr.ResponseWriter, "Would send email for %s to %s.<br>", p.Person.Email, wr.LoginInfo.Person.Email)
		}
	}
	return nil
}

func AttendeesDryRunDistributor(wr WrappedRequest, sender EmailSender) error {
	wr.ResponseWriter.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(wr.ResponseWriter, "Looking up all attendees...<br>")

	q := datastore.NewQuery("Invitation").Filter("Event =", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := q.GetAll(wr.Context, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(wr.Context, invitationKeys[i],
			invitations[i])
		roomingInfo := getRoomingInfoWithInvitation(wr, invitations[i], invitationKeys[i])
		if roomingInfo == nil {
			continue
		}
		for _, p := range realizedInvitation.Invitees {
			if p.Person.Email == "" {
				continue
			}
			if _, found := roomingInfo.Attendees[p.Person.DatastoreKey.IntID()]; !found {
				continue
			}
			emailData := map[string]interface{}{
				"Event":       wr.Event,
				"Invitation":  realizedInvitation,
				"Person":      &p.Person,
				"RoomingInfo": roomingInfo,
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

func AttendeesDistributor(wr WrappedRequest, sender EmailSender) error {
	wr.ResponseWriter.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(wr.ResponseWriter, "Looking up all attendees...<br>")

	q := datastore.NewQuery("Invitation").Filter("Event =", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := q.GetAll(wr.Context, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(wr.Context, invitationKeys[i],
			invitations[i])
		roomingInfo := getRoomingInfoWithInvitation(wr, invitations[i], invitationKeys[i])
		if roomingInfo == nil {
			continue
		}
		for _, p := range realizedInvitation.Invitees {
			if p.Person.Email == "" {
				continue
			}
			if _, found := roomingInfo.Attendees[p.Person.DatastoreKey.IntID()]; !found {
				continue
			}
			emailData := map[string]interface{}{
				"Event":       wr.Event,
				"Invitation":  realizedInvitation,
				"Person":      &p.Person,
				"RoomingInfo": roomingInfo,
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
