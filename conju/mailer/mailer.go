package mailer

import (
	"context"
	"fmt"
	"strings"
)

type Client interface {
	Send(ctx context.Context, msg *Message) error
}

func FromContext(ctx context.Context) (Client, error) {
	c, ok := ctx.Value(mailerKey{}).(Client)
	if !ok {
		return nil, fmt.Errorf("no email client in context")
	}
	return c, nil
}

func WrapContext(ctx context.Context, client Client) context.Context {
	return context.WithValue(ctx, mailerKey{}, client)
}

type mailerKey struct{}

// An email address, generally rendered as Name <Addr>
type Email struct {
	Name string
	Addr string
}

func (e Email) String() string {
	if e.Name == "" {
		return "<" + e.Addr + ">"
	}
	return e.Name + " <" + e.Addr + ">"
}

func CommaSeparated(emails []Email) string {
	if len(emails) == 0 {
		return ""
	}
	var buf strings.Builder
	for i, e := range emails {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(e.String())
	}
	return buf.String()
}

// Generic message to be sent.
type Message struct {
	From    Email
	To      Email
	CC      []Email
	BCC     []Email
	Subject string
	Text    string
	HTML    string
}
