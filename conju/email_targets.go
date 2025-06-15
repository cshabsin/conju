package conju

import (
	"context"
	"fmt"

	"cloud.google.com/go/datastore"
	"github.com/cshabsin/conju/conju/dsclient"
	"github.com/cshabsin/conju/invitation"
)

// This file defines a set of EmailDistributors, which the
// handleSendMail function uses to

type EmailSender func(context.Context, map[string]interface{}, MailHeaderInfo) error

type EmailDistributor func(context.Context, WrappedRequest, EmailSender) error
type EmailDistributorEntry struct {
	NeedsConfirm bool
	Distribute   EmailDistributor
}

var AllDistributors = map[string]EmailDistributorEntry{
	"SelfOnly":              {false, SelfOnlyDistributor},
	"AllInviteesDryRun":     {false, AllInviteesDryRunDistributor},
	"AllInvitees*REAL*":     {true, AllInviteesDistributor},
	"AttendeesList":         {false, AttendeesListDistributor},
	"AttendeesDryRun":       {false, AttendeesDryRunDistributor},
	"Attendees*REAL*":       {true, AttendeesDistributor},
	"QualifiedInviteesList": {false, QualifiedInviteesListDistributor},
	"QualifiedDryRunList":   {false, QualifiedInviteesDryRunDistributor},
	"Qualified*REAL*":       {true, QualifiedInviteesDistributor},
}

func SelfOnlyDistributor(ctx context.Context, wr WrappedRequest, sender EmailSender) error {
	client := dsclient.FromContext(ctx)
	if client == nil {
		return fmt.Errorf("datastore client is nil")
	}
	wr.ResponseWriter.Header().Set("Content-Type", "text/html")
	realizedInvitation := makeRealizedInvitation(ctx, wr.LoginInfo.InvitationKey,
		wr.LoginInfo.Invitation)
	roomingInfo := getRoomingInfoWithInvitation(ctx, wr, wr.LoginInfo.Invitation, wr.LoginInfo.InvitationKey)
	fmt.Fprintf(wr.ResponseWriter, "Sending only to &lt;%s&gt;.<br>", wr.LoginInfo.Person.Email)
	emailData := map[string]interface{}{
		"Event":       wr.Event,
		"Invitation":  realizedInvitation,
		"Person":      wr.LoginInfo.Person,
		"RoomingInfo": roomingInfo,
	}
	err := sender(ctx, emailData, MailHeaderInfo{To: []string{wr.LoginInfo.Person.Email}})
	return err
}

func AllInviteesDryRunDistributor(ctx context.Context, wr WrappedRequest, sender EmailSender) error {
	wr.ResponseWriter.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(wr.ResponseWriter, "Looking up all invitees...<br>")

	q := datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(ctx, invitationKeys[i],
			invitations[i])
		roomingInfo := getRoomingInfoWithInvitation(ctx, wr, invitations[i], invitationKeys[i])
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
			err := sender(ctx, emailData, MailHeaderInfo{To: []string{wr.LoginInfo.Person.Email}})
			if err != nil {
				fmt.Fprintf(wr.ResponseWriter, "Error emailing %s: %v", p.Person.Email, err)
				return err
			}
		}
	}
	return nil
}

func AllInviteesDistributor(ctx context.Context, wr WrappedRequest, sender EmailSender) error {
	wr.ResponseWriter.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(wr.ResponseWriter, "Looking up all invitees...<br>")

	q := datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(ctx, invitationKeys[i],
			invitations[i])
		roomingInfo := getRoomingInfoWithInvitation(ctx, wr, invitations[i], invitationKeys[i])
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
			err := sender(ctx, emailData, MailHeaderInfo{To: []string{p.Person.Email}})
			if err != nil {
				fmt.Fprintf(wr.ResponseWriter, "Error emailing %s: %v", p.Person.Email, err)
				return err
			}
		}
	}
	return nil
}

func AttendeesListDistributor(ctx context.Context, wr WrappedRequest, sender EmailSender) error {
	wr.ResponseWriter.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(wr.ResponseWriter, "Looking up all attendees...<br>")

	q := datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(ctx, invitationKeys[i],
			invitations[i])
		roomingInfo := getRoomingInfoWithInvitation(ctx, wr, invitations[i], invitationKeys[i])
		if roomingInfo == nil {
			continue
		}
		for _, p := range realizedInvitation.Invitees {
			if p.Person.Email == "" {
				continue
			}
			if _, found := roomingInfo.Attendees[p.Person.DatastoreKey.ID]; !found {
				continue
			}
			fmt.Fprintf(wr.ResponseWriter, "Would send email for %s to %s.<br>", p.Person.Email, wr.LoginInfo.Person.Email)
		}
	}
	return nil
}

func AttendeesDryRunDistributor(ctx context.Context, wr WrappedRequest, sender EmailSender) error {
	wr.ResponseWriter.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(wr.ResponseWriter, "Looking up all attendees...<br>")

	q := datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(ctx, invitationKeys[i],
			invitations[i])
		roomingInfo := getRoomingInfoWithInvitation(ctx, wr, invitations[i], invitationKeys[i])
		if roomingInfo == nil {
			continue
		}
		for _, p := range realizedInvitation.Invitees {
			if p.Person.Email == "" {
				continue
			}
			if _, found := roomingInfo.Attendees[p.Person.DatastoreKey.ID]; !found {
				continue
			}
			emailData := map[string]interface{}{
				"Event":       wr.Event,
				"Invitation":  realizedInvitation,
				"Person":      &p.Person,
				"RoomingInfo": roomingInfo,
			}
			fmt.Fprintf(wr.ResponseWriter, "Sending email for %s to %s.<br>", p.Person.Email, wr.LoginInfo.Person.Email)
			err := sender(ctx, emailData, MailHeaderInfo{To: []string{wr.LoginInfo.Person.Email}})
			if err != nil {
				fmt.Fprintf(wr.ResponseWriter, "Error emailing %s: %v", p.Person.Email, err)
				return err
			}
		}
	}
	return nil
}

func AttendeesDistributor(ctx context.Context, wr WrappedRequest, sender EmailSender) error {
	wr.ResponseWriter.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(wr.ResponseWriter, "Looking up all attendees...<br>")

	q := datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(ctx, invitationKeys[i],
			invitations[i])
		roomingInfo := getRoomingInfoWithInvitation(ctx, wr, invitations[i], invitationKeys[i])
		if roomingInfo == nil {
			continue
		}
		for _, p := range realizedInvitation.Invitees {
			if p.Person.Email == "" {
				continue
			}
			if _, found := roomingInfo.Attendees[p.Person.DatastoreKey.ID]; !found {
				continue
			}
			emailData := map[string]interface{}{
				"Event":       wr.Event,
				"Invitation":  realizedInvitation,
				"Person":      &p.Person,
				"RoomingInfo": roomingInfo,
			}
			fmt.Fprintf(wr.ResponseWriter, "Sending email for %s.<br>", p.Person.Email)
			err := sender(ctx, emailData, MailHeaderInfo{To: []string{p.Person.Email}})
			if err != nil {
				fmt.Fprintf(wr.ResponseWriter, "Error emailing %s: %v", p.Person.Email, err)
				return err
			}
		}
	}
	return nil
}

// QualifiedInviteesListDistributor is an email distributor that lists all invitees
// who have not RSVP'ed "no" to the event. If RsvpMap is nil, the invitee has not
// submitted any RSVP at all, and the person is included.
func QualifiedInviteesListDistributor(ctx context.Context, wr WrappedRequest, sender EmailSender) error {
	wr.ResponseWriter.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(wr.ResponseWriter, "Looking up all invitees...<br>")

	q := datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(ctx, invitationKeys[i],
			invitations[i])
		for _, p := range realizedInvitation.Invitees {
			if p.Person.Email == "" {
				continue
			}
			if len(realizedInvitation.RsvpMap) != 0 && realizedInvitation.RsvpMap[p.Key].Status == invitation.No {
				fmt.Fprintf(wr.ResponseWriter, "Skipping recipient %s: %v<br>", p.Person.Email, realizedInvitation.RsvpMap[p.Key].Status)
				continue
			}
			fmt.Fprintf(wr.ResponseWriter, "Would send email for %s to %s.<br>", p.Person.Email, wr.LoginInfo.Person.Email)
		}
	}
	return nil
}

// QualifiedInviteesDryRunDistributor is an email distributor that sends the currently
// logged in user one email for each person who has not RSVP'ed "no" to the event.
func QualifiedInviteesDryRunDistributor(ctx context.Context, wr WrappedRequest, sender EmailSender) error {
	wr.ResponseWriter.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(wr.ResponseWriter, "Looking up all invitees...<br>")

	q := datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(ctx, invitationKeys[i],
			invitations[i])
		roomingInfo := getRoomingInfoWithInvitation(ctx, wr, invitations[i], invitationKeys[i])
		for _, p := range realizedInvitation.Invitees {
			if p.Person.Email == "" {
				continue
			}
			if len(realizedInvitation.RsvpMap) != 0 && realizedInvitation.RsvpMap[p.Key].Status == invitation.No {
				continue
			}
			emailData := map[string]interface{}{
				"Event":       wr.Event,
				"Invitation":  realizedInvitation,
				"Person":      &p.Person,
				"RoomingInfo": roomingInfo,
			}
			fmt.Fprintf(wr.ResponseWriter, "Would send email for %s to %s.<br>", p.Person.Email, wr.LoginInfo.Person.Email)
			err := sender(ctx, emailData, MailHeaderInfo{To: []string{wr.LoginInfo.Person.Email}})
			if err != nil {
				fmt.Fprintf(wr.ResponseWriter, "Error emailing %s: %v", p.Person.Email, err)
				return err
			}
		}
	}
	return nil
}

// QualifiedInviteesDistributor is an email distributor that sends an email
// to each person who has not RSVP'ed "no" to the event.
func QualifiedInviteesDistributor(ctx context.Context, wr WrappedRequest, sender EmailSender) error {
	wr.ResponseWriter.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(wr.ResponseWriter, "Looking up all invitees...<br>")

	q := datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(ctx, invitationKeys[i],
			invitations[i])
		roomingInfo := getRoomingInfoWithInvitation(ctx, wr, invitations[i], invitationKeys[i])
		for _, p := range realizedInvitation.Invitees {
			if p.Person.Email == "" {
				continue
			}
			if len(realizedInvitation.RsvpMap) != 0 && realizedInvitation.RsvpMap[p.Key].Status == invitation.No {
				continue
			}
			emailData := map[string]interface{}{
				"Event":       wr.Event,
				"Invitation":  realizedInvitation,
				"Person":      &p.Person,
				"RoomingInfo": roomingInfo,
			}
			fmt.Fprintf(wr.ResponseWriter, "Would send email for %s to %s.<br>", p.Person.Email, wr.LoginInfo.Person.Email)
			err := sender(ctx, emailData, MailHeaderInfo{To: []string{p.Person.Email}})
			if err != nil {
				fmt.Fprintf(wr.ResponseWriter, "Error emailing %s: %v", p.Person.Email, err)
				return err
			}
		}
	}
	return nil
}
