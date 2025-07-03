package forwardemail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/cshabsin/conju/conju/backends/secretmanager"
	"github.com/cshabsin/conju/conju/mailer"
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

func NewClientFromSecretManager(ctx context.Context, client *secretmanager.Client) (*Client, error) {
	apiKey, err := client.Get(ctx, "forwardemail_api_key")
	if err != nil {
		return nil, fmt.Errorf("error getting api key for forwardemail: %w", err)
	}
	return &Client{apiKey: apiKey}, nil
}

func (c *Client) Send(ctx context.Context, m *mailer.Message) error {
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
	_, err = io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return nil
}
