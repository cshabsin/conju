package sendgrid

import (
	"context"
	"fmt"

	"github.com/cshabsin/conju/conju/backends/secretmanager"
	"github.com/cshabsin/conju/conju/mailer"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// NewClient creates a new Sendgrid client, retrieving the API key from the secretmanager client
// in the context.
func NewClient(ctx context.Context) (*Client, error) {
	secretclient, err := secretmanager.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	apiKey, err := secretclient.Get(ctx, "sendgrid_api_key")
	if err != nil {
		return nil, err
	}
	return &Client{client: sendgrid.NewSendClient(string(apiKey))}, nil
}

type Client struct {
	client *sendgrid.Client
}

var _ mailer.Client = &Client{}

func (c *Client) Send(ctx context.Context, msg *mailer.Message) error {
	m := mail.NewSingleEmail(
		toSGEmail(msg.From),
		msg.Subject,
		toSGEmail(msg.To),
		msg.Text,
		msg.HTML,
	)
	if len(msg.CC) > 0 || len(msg.BCC) > 0 {
		personalization := mail.NewPersonalization()
		personalization.AddTos(toSGEmail(msg.To))
		for _, cc := range msg.CC {
			personalization.AddCCs(toSGEmail(cc))
		}
		for _, bcc := range msg.BCC {
			personalization.AddBCCs(toSGEmail(bcc))
		}
		personalization.Subject = msg.Subject
		m.AddPersonalizations(personalization)
	}

	// Send the email
	response, err := c.client.Send(m)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	// Check if the response indicates an error
	if response.StatusCode >= 400 {
		return fmt.Errorf("sendgrid returned error status %d: %s", response.StatusCode, response.Body)
	}

	return nil
}

func toSGEmail(e mailer.Email) *mail.Email {
	return mail.NewEmail(e.Name, e.Addr)
}
