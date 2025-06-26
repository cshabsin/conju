// Package message provides the definition of the Message struct and its methods.
//
// Message represents an email message template that can be mailed to users.
// Primary keys:
//
// - event
// - email shortname (currently the filename in the repo).
//
// Columns:
//
// - a type enum, to determine which recipients are targeted by the email (individuals, invitees, or attendees)
// - plaintext version of the message template (using golang tmpl language - include a "cheat sheet" on the edit page?)
// - html version of the message template
package message

import (
	"context"
	"fmt"
	html_template "html/template"
	"strings"
	text_template "text/template"

	"cloud.google.com/go/datastore"
	"github.com/cshabsin/conju/conju/dsclient"
	"google.golang.org/api/iterator"
)

type MessageType int

const (
	// IndividualMessage is a message sent to an individual user.
	IndividualMessage MessageType = 1
	// InviteeMessage is a message sent to an invitee of an event.
	InviteeMessage MessageType = 2
	// AttendeeMessage is a message sent to an attendee of an event.
	AttendeeMessage MessageType = 3
	// AdminMessage is a message sent to an admin of an event.
	AdminMessage MessageType = 4
)

// Message represents an email message template that can be mailed to users.
type Message struct {
	Event     *datastore.Key // the event id this message is associated with
	ShortName string         // the short name of the message, used to identify it

	Type      MessageType // the type of message (individual, invitee, attendee, admin)
	Subject   string      `datastore:",noindex"` // subject template for the message
	Plaintext string      `datastore:",noindex"` // plaintext version of the message template
	HTML      string      `datastore:",noindex"` // HTML version of the message template

	// Whether this message can be selected by users for sending.
	// If false, it just contains utility templates that are used by other messages.
	Selectable bool
}

// Key returns the datastore key for the message.
func (m *Message) Key() *datastore.Key {
	return datastore.NameKey("Message", m.ShortName, m.Event)
}

func forEvent(ctx context.Context, eventKey *datastore.Key) ([]*Message, error) {
	client := dsclient.FromContext(ctx)
	q := datastore.NewQuery("Message").FilterField("Event", "=", eventKey)
	var messages []*Message
	_, err := client.GetAll(ctx, q, &messages)
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func global(ctx context.Context) ([]*Message, error) {
	client := dsclient.FromContext(ctx)
	var noEvent *datastore.Key // nil key indicates global messages
	q := datastore.NewQuery("Message").FilterField("Event", "=", noEvent).Order("ShortName")
	var globalMessages []*Message
	_, err := client.GetAll(ctx, q, &globalMessages)
	if err != nil {
		return nil, err
	}
	return globalMessages, nil
}

type MessageInfo struct {
	Key       *datastore.Key // the datastore key of the message
	EventKey  *datastore.Key // true if this message is associated with an event
	ShortName string         // the short name of the message
}

func ListTemplates(ctx context.Context, eventKey *datastore.Key) ([]MessageInfo, error) {
	client := dsclient.FromContext(ctx)
	q := datastore.NewQuery("Message")
	var msgs []*Message
	keys, err := client.GetAll(ctx, q, &msgs)
	if err != nil {
		return nil, fmt.Errorf("error querying templates: %w", err)
	}

	var names []MessageInfo
	for i, msg := range msgs {
		if msg.Selectable {
			names = append(names, MessageInfo{Key: keys[i], EventKey: msg.Event, ShortName: msg.ShortName})
		}
	}
	return names, nil
}

func FetchTemplateSource(ctx context.Context, eventKey *datastore.Key, name string) (*Message, error) {
	client := dsclient.FromContext(ctx)
	q := datastore.NewQuery("Message").FilterField("Event", "=", eventKey).FilterField("ShortName", "=", name)

	it := client.Run(ctx, q)

	var msg Message
	_, err := it.Next(&msg)
	if err == iterator.Done {
		return nil, fmt.Errorf("message %q not found", name)
	}
	return &msg, nil
}

func SaveTemplate(ctx context.Context, msg *Message) error {
	if msg.ShortName == "" {
		return fmt.Errorf("message must have a short name")
	}
	client := dsclient.FromContext(ctx)
	if _, err := client.Put(ctx, msg.Key(), msg); err != nil {
		return fmt.Errorf("error saving message %q: %w", msg.ShortName, err)
	}
	return nil
}

func Get(ctx context.Context, key *datastore.Key) (*Message, error) {
	client := dsclient.FromContext(ctx)
	var msg Message
	if err := client.Get(ctx, key, &msg); err != nil {
		return nil, fmt.Errorf("error retrieving message %q: %w", key.Encode(), err)
	}
	return &msg, nil
}

func GetTemplates(ctx context.Context, eventKey *datastore.Key, funcMap text_template.FuncMap) (*html_template.Template, *text_template.Template, error) {
	msgs, err := global(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("error listing global templates: %w", err)
	}
	eventMsgs, err := forEvent(ctx, eventKey)
	if err != nil {
		return nil, nil, fmt.Errorf("error listing event templates: %w", err)
	}
	msgs = append(msgs, eventMsgs...)
	htmlTpl := html_template.New("").Funcs(funcMap)
	txtTpl := text_template.New("").Funcs(funcMap)
	for _, msg := range msgs {
		if _, err := htmlTpl.Parse(htmlContents(msg)); err != nil {
			return nil, nil, fmt.Errorf("error parsing HTML template %q: %w", msg.ShortName, err)
		}
		if _, err := txtTpl.Parse(textContents(msg)); err != nil {
			return nil, nil, fmt.Errorf("error parsing text template %q: %w", msg.ShortName, err)
		}
	}
	return htmlTpl, txtTpl, nil
}

func htmlContents(m *Message) string {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("{{define \"%s_subject\"}}\n%s\n{{end}}\n", m.ShortName, m.Subject))
	content.WriteString(fmt.Sprintf("{{define \"%s_html\"}}\n%s\n{{end}}\n", m.ShortName, m.HTML))
	return content.String()
}

func textContents(m *Message) string {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("{{define \"%s_subject\"}}\n%s\n{{end}}\n", m.ShortName, m.Subject))
	content.WriteString(fmt.Sprintf("{{define \"%s_text\"}}\n%s\n{{end}}\n", m.ShortName, m.Plaintext))
	return content.String()
}
