package conju

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/datastore"
	"github.com/cshabsin/conju/conju/dsclient"
	"github.com/cshabsin/conju/conju/mailer"
	"github.com/cshabsin/conju/invitation"
	"github.com/cshabsin/conju/model/person"
	"github.com/gin-gonic/gin"
)

// This file defines a set of EmailDistributors, which the
// handleSendMail function uses to

// MailHeaderInfo contains the header info for outgoing email, passed into sendMail.
type MailHeaderInfo struct {
	To      mailer.Email
	Cc      []mailer.Email
	Bcc     []mailer.Email
	Subject string

	BccSelf bool
}

type EmailSender func(context.Context, map[string]any, MailHeaderInfo) error

type EmailDistributor func(*gin.Context, EmailSender) error
type EmailDistributorEntry struct {
	NeedsConfirm bool
	Distribute   EmailDistributor
}

var AllDistributors = map[string]EmailDistributorEntry{
	"SelfOnly":              {false, SelfOnlyDistributor},
	"AllInviteesDryRun0":    {false, AllInviteesDryRunDistributor(0)},
	"AllInvitees*REAL*0":    {true, AllInviteesDistributor(0)},
	"AllInviteesDryRun1":    {false, AllInviteesDryRunDistributor(1)},
	"AllInvitees*REAL*1":    {true, AllInviteesDistributor(1)},
	"AllInviteesDryRun2":    {false, AllInviteesDryRunDistributor(2)},
	"AllInvitees*REAL*2":    {true, AllInviteesDistributor(2)},
	"AllInviteesList0":      {false, AllInviteesListDistributor(0)},
	"AllInviteesList1":      {false, AllInviteesListDistributor(1)},
	"AllInviteesList2":      {false, AllInviteesListDistributor(2)},
	"AttendeesList0":        {false, AttendeesListDistributor(0)},
	"AttendeesList1":        {false, AttendeesListDistributor(1)},
	"AttendeesList2":        {false, AttendeesListDistributor(2)},
	"AttendeesDryRun":       {false, AttendeesDryRunDistributor},
	"Attendees*REAL*":       {true, AttendeesDistributor},
	"QualifiedInviteesList": {false, QualifiedInviteesListDistributor},
	"QualifiedDryRunList":   {false, QualifiedInviteesDryRunDistributor},
	"Qualified*REAL*":       {true, QualifiedInviteesDistributor},
}

func EmailForPerson(p *person.Person) mailer.Email {
	return mailer.Email{
		Name: p.FullName(),
		Addr: p.Email,
	}
}

func SelfOnlyDistributor(c *gin.Context, sender EmailSender) error {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	ctx := c.Request.Context()
	c.Header("Content-Type", "text/html")
	realizedInvitation := makeRealizedInvitation(ctx, wr.LoginInfo.InvitationKey,
		wr.LoginInfo.Invitation)
	roomingInfo := GetRoomingInfoWithInvitation(ctx, wr.GetBookingInfo(ctx), wr.Event, wr.LoginInfo.Invitation, wr.LoginInfo.InvitationKey)
	c.String(200, "Sending only to &lt;%s&gt;.<br>", wr.LoginInfo.Person.Email)
	emailData := map[string]any{
		"Event":       wr.Event,
		"Invitation":  realizedInvitation,
		"Person":      wr.LoginInfo.Person,
		"RoomingInfo": roomingInfo,
	}
	err := sender(ctx, emailData, MailHeaderInfo{To: EmailForPerson(wr.LoginInfo.Person)})
	return err
}

func AllInviteesDryRunDistributor(tier int) EmailDistributor {
	return func(c *gin.Context, sender EmailSender) error {
		return AllInviteesDryRunDistributorImpl(c, sender, tier)
	}
}

func AllInviteesDryRunDistributorImpl(c *gin.Context, sender EmailSender, tier int) error {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	ctx := c.Request.Context()
	c.Header("Content-Type", "text/html")
	c.String(200, "Looking up all invitees...<br>")

	q := datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(ctx, invitationKeys[i],
			invitations[i])
		roomingInfo := GetRoomingInfoWithInvitation(ctx, wr.GetBookingInfo(ctx), wr.Event, invitations[i], invitationKeys[i])
		for _, p := range realizedInvitation.Invitees {
			if p.Person.Email == "" {
				continue
			}
			if p.Person.EmailTier != tier {
				continue
			}
			emailData := map[string]any{
				"Event":       wr.Event,
				"Invitation":  realizedInvitation,
				"Person":      &p.Person,
				"RoomingInfo": roomingInfo,
			}
			c.String(200, "Sending email for %s to %s.<br>", p.Person.Email, wr.LoginInfo.Person.Email)
			err := sender(ctx, emailData, MailHeaderInfo{To: EmailForPerson(wr.LoginInfo.Person)})
			if err != nil {
				c.String(500, "Error emailing %s: %v", p.Person.Email, err)
				return err
			}
		}
	}
	return nil
}

func AllInviteesDistributor(tier int) EmailDistributor {
	return func(c *gin.Context, sender EmailSender) error {
		return AllInviteesDistributorImpl(c, sender, tier)
	}
}

func AllInviteesDistributorImpl(c *gin.Context, sender EmailSender, tier int) error {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	ctx := c.Request.Context()
	c.Header("Content-Type", "text/html")
	c.String(200, "Looking up all invitees...<br>")

	q := datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(ctx, invitationKeys[i],
			invitations[i])
		roomingInfo := GetRoomingInfoWithInvitation(ctx, wr.GetBookingInfo(ctx), wr.Event, invitations[i], invitationKeys[i])
		for _, p := range realizedInvitation.Invitees {
			if p.Person.Email == "" {
				continue
			}
			if p.Person.EmailTier != tier {
				continue
			}
			emailData := map[string]any{
				"Event":       wr.Event,
				"Invitation":  realizedInvitation,
				"Person":      &p.Person,
				"RoomingInfo": roomingInfo,
			}
			c.String(200, "Sending email for %s.<br>", p.Person.Email)
			err := sender(ctx, emailData, MailHeaderInfo{To: EmailForPerson(&p.Person)})
			if err != nil {
				c.String(500, "Error emailing %s: %v", p.Person.Email, err)
				return err
			}
		}
	}
	return nil
}

func AllInviteesListDistributor(tier int) EmailDistributor {
	return func(c *gin.Context, sender EmailSender) error {
		return AllInviteesListDistributorImpl(c, sender, tier)
	}
}

func AllInviteesListDistributorImpl(c *gin.Context, sender EmailSender, tier int) error {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	ctx := c.Request.Context()
	c.Header("Content-Type", "text/html")
	c.String(200, "Looking up all invitees...<br>")

	q := datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := MakeRealizedInvitation(ctx, invitationKeys[i],
			invitations[i])
		for _, p := range realizedInvitation.Invitees {
			if p.Person.Email == "" {
				continue
			}
			if p.Person.EmailTier != tier {
				c.String(200, "Skipping %s (tier %d).<br>", p.Person.Email, p.Person.EmailTier)
				continue
			}
			c.String(200, "Would send email for %s.<br>", p.Person.Email)
		}
	}
	return nil
}

func AttendeesListDistributor(tier int) EmailDistributor {
	return func(c *gin.Context, sender EmailSender) error {
		log.Printf("AttendeesListDistributor called with tier %d", tier)
		return AttendeesListDistributorImpl(c, sender, tier)
	}
}

func AttendeesListDistributorImpl(c *gin.Context, sender EmailSender, tier int) error {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	ctx := c.Request.Context()
	c.Header("Content-Type", "text/html")
	c.String(200, "Looking up all attendees...<br>")

	q := datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(ctx, invitationKeys[i],
			invitations[i])
		roomingInfo := GetRoomingInfoWithInvitation(ctx, wr.GetBookingInfo(ctx), wr.Event, invitations[i], invitationKeys[i])
		if roomingInfo == nil {
			continue
		}
		for _, p := range realizedInvitation.Invitees {
			if p.Person.Email == "" {
				continue
			}
			if p.Person.EmailTier != tier {
				continue
			}
			if _, found := roomingInfo.Attendees[p.Person.DatastoreKey.ID]; !found {
				continue
			}
			c.String(200, "Would send email for %s to %s.<br>", p.Person.Email, wr.LoginInfo.Person.Email)
		}
	}
	return nil
}

func AttendeesDryRunDistributor(c *gin.Context, sender EmailSender) error {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	ctx := c.Request.Context()
	c.Header("Content-Type", "text/html")
	c.String(200, "Looking up all attendees...<br>")

	q := datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(ctx, invitationKeys[i],
			invitations[i])
		roomingInfo := GetRoomingInfoWithInvitation(ctx, wr.GetBookingInfo(ctx), wr.Event, invitations[i], invitationKeys[i])
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
			emailData := map[string]any{
				"Event":       wr.Event,
				"Invitation":  realizedInvitation,
				"Person":      &p.Person,
				"RoomingInfo": roomingInfo,
			}
			c.String(200, "Sending email for %s to %s.<br>", p.Person.Email, wr.LoginInfo.Person.Email)
			err := sender(ctx, emailData, MailHeaderInfo{To: EmailForPerson(wr.LoginInfo.Person)})
			if err != nil {
				c.String(500, "Error emailing %s: %v", p.Person.Email, err)
				return err
			}
		}
	}
	return nil
}

func AttendeesDistributor(c *gin.Context, sender EmailSender) error {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	ctx := c.Request.Context()
	c.Header("Content-Type", "text/html")
	c.String(200, "Looking up all attendees...<br>")

	q := datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(ctx, invitationKeys[i],
			invitations[i])
		roomingInfo := GetRoomingInfoWithInvitation(ctx, wr.GetBookingInfo(ctx), wr.Event, invitations[i], invitationKeys[i])
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
			emailData := map[string]any{
				"Event":       wr.Event,
				"Invitation":  realizedInvitation,
				"Person":      &p.Person,
				"RoomingInfo": roomingInfo,
			}
			c.String(200, "Sending email for %s.<br>", p.Person.Email)
			err := sender(ctx, emailData, MailHeaderInfo{To: EmailForPerson(&p.Person)})
			if err != nil {
				c.String(500, "Error emailing %s: %v", p.Person.Email, err)
				return err
			}
		}
	}
	return nil
}

// QualifiedInviteesListDistributor is an email distributor that lists all invitees
// who have not RSVP'ed "no" to the event. If RsvpMap is nil, the invitee has not
// submitted any RSVP at all, and the person is included.
func QualifiedInviteesListDistributor(c *gin.Context, sender EmailSender) error {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	ctx := c.Request.Context()
	c.Header("Content-Type", "text/html")
	c.String(200, "Looking up all invitees...<br>")

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
				c.String(200, "Skipping recipient %s: %v<br>", p.Person.Email, realizedInvitation.RsvpMap[p.Key].Status)
				continue
			}
			c.String(200, "Would send email for %s to %s.<br>", p.Person.Email, wr.LoginInfo.Person.Email)
		}
	}
	return nil
}

// QualifiedInviteesDryRunDistributor is an email distributor that sends the currently
// logged in user one email for each person who has not RSVP'ed "no" to the event.
func QualifiedInviteesDryRunDistributor(c *gin.Context, sender EmailSender) error {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	ctx := c.Request.Context()
	c.Header("Content-Type", "text/html")
	c.String(200, "Looking up all invitees...<br>")

	q := datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(ctx, invitationKeys[i],
			invitations[i])
		roomingInfo := GetRoomingInfoWithInvitation(ctx, wr.GetBookingInfo(ctx), wr.Event, invitations[i], invitationKeys[i])
		for _, p := range realizedInvitation.Invitees {
			if p.Person.Email == "" {
				continue
			}
			if len(realizedInvitation.RsvpMap) != 0 && realizedInvitation.RsvpMap[p.Key].Status == invitation.No {
				continue
			}
			emailData := map[string]any{
				"Event":       wr.Event,
				"Invitation":  realizedInvitation,
				"Person":      &p.Person,
				"RoomingInfo": roomingInfo,
			}
			c.String(200, "Would send email for %s to %s.<br>", p.Person.Email, wr.LoginInfo.Person.Email)
			err := sender(ctx, emailData, MailHeaderInfo{To: EmailForPerson(wr.LoginInfo.Person)})
			if err != nil {
				c.String(500, "Error emailing %s: %v", p.Person.Email, err)
				return err
			}
		}
	}
	return nil
}

// QualifiedInviteesDistributor is an email distributor that sends an email
// to each person who has not RSVP'ed "no" to the event.
func QualifiedInviteesDistributor(c *gin.Context, sender EmailSender) error {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	ctx := c.Request.Context()
	c.Header("Content-Type", "text/html")
	c.String(200, "Looking up all invitees...<br>")

	q := datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	var invitations []*Invitation
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		return err
	}
	for i := 0; i < len(invitations); i++ {
		realizedInvitation := makeRealizedInvitation(ctx, invitationKeys[i],
			invitations[i])
		roomingInfo := GetRoomingInfoWithInvitation(ctx, wr.GetBookingInfo(ctx), wr.Event, invitations[i], invitationKeys[i])
		for _, p := range realizedInvitation.Invitees {
			if p.Person.Email == "" {
				continue
			}
			if len(realizedInvitation.RsvpMap) != 0 && realizedInvitation.RsvpMap[p.Key].Status == invitation.No {
				continue
			}
			emailData := map[string]any{
				"Event":       wr.Event,
				"Invitation":  realizedInvitation,
				"Person":      &p.Person,
				"RoomingInfo": roomingInfo,
			}
			c.String(200, "Would send email for %s to %s.<br>", p.Person.Email, wr.LoginInfo.Person.Email)
			err := sender(ctx, emailData, MailHeaderInfo{To: EmailForPerson(&p.Person)})
			if err != nil {
				c.String(500, "Error emailing %s: %v", p.Person.Email, err)
				return err
			}
		}
	}
	return nil
}