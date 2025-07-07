// Package forwardemail is a client for forwardemail.net.
//
// It uses the APIs documented at https://forwardemail.net/en/email-api.
//
// It retrieves the API key from the secretmanager client using the client
// stashed in the context by secretmanager.WrapContext.
//
// Store the API key as the 26-character "API token" as viewed in the Security
// section of the forwardemail.net account settings. The code manages the
// proper base64 encoding of it as a username in their weird setup.
package forwardemail

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/cshabsin/conju/conju/mailer"
	"github.com/cshabsin/conju/conju/secretmanager"
)

type Client struct {
	apiKey string
}

var _ mailer.Client = &Client{}

type Message struct {
	From    string `json:"from"`
	To      string `json:"to"`
	CC      string `json:"cc"`
	BCC     string `json:"bcc"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
	HTML    string `json:"html"`
}

func NewClientFromKey(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
	}
}

func NewClientFromEnv() *Client {
	return NewClientFromKey(os.Getenv("FORWARDEMAIL_API_KEY"))
}

func NewClientFromSecretManager(ctx context.Context, secretClient *secretmanager.Client) (*Client, error) {
	client := &Client{}
	if err := client.RefreshSecret(ctx, secretClient); err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Client) RefreshSecret(ctx context.Context, secretClient *secretmanager.Client) error {
	keyBytes, err := secretClient.Get(ctx, "forwardemail_api_key")
	if err != nil {
		return fmt.Errorf("error getting api key for forwardemail: %w", err)
	}
	apiKey := string(keyBytes)
	apiKey = base64.StdEncoding.EncodeToString([]byte(apiKey + ":"))
	c.apiKey = apiKey
	return nil
}

func (c *Client) Send(ctx context.Context, m *mailer.Message) error {
	log.Printf("forwardemail sending email to %s from %s", m.To.String(), m.From.String())
	url := "https://api.forwardemail.net/v1/emails"

	msg := &Message{
		To:      m.To.String(),
		From:    m.From.String(),
		Subject: m.Subject,
		Text:    m.Text,
		HTML:    m.HTML,
		CC:      mailer.CommaSeparated(m.CC),
		BCC:     mailer.CommaSeparated(m.BCC),
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(msg); err != nil {
		return fmt.Errorf("error encoding message: %w", err)
	}

	req, _ := http.NewRequestWithContext(ctx, "POST", url, &buf)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Basic "+c.apiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode >= 400 {
		return fmt.Errorf("forwardemail returned status %d: %s", res.StatusCode, string(body))
	}

	return nil
}
