package sendgrid

import (
	"context"
	"fmt"

	"github.com/cshabsin/conju/conju/backends/secretmanager"
	cmail "github.com/cshabsin/conju/conju/mail"
	"github.com/sendgrid/sendgrid-go"
	sgmail "github.com/sendgrid/sendgrid-go/helpers/mail"
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
	return &Client{client: sendgrid.NewSendClient(apiKey)}, nil
}

type Client struct {
	client *sendgrid.Client
}

var _ cmail.Client = &Client{}

func (c *Client) Send(ctx context.Context, msg *cmail.Message) error {
	m := sgmail.NewSingleEmail(
		toSGEmail(msg.From),
		msg.Subject,
		toSGEmail(msg.To),
		msg.Text,
		msg.HTML,
	)
	if len(msg.CC) > 0 || len(msg.BCC) > 0 {
		personalization := sgmail.NewPersonalization()
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

func toSGEmail(e cmail.Email) *sgmail.Email {
	return sgmail.NewEmail(e.Name, e.Addr)
}
