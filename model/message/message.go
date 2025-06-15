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

import "cloud.google.com/go/datastore"

type Message struct {
	Event     *datastore.Key // the event id this message is associated with
	ShortName string         // the short name of the message, used to identify it
}
