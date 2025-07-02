package forwardemail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Client struct {
	apiKey string
}

func NewClientFromKey(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
	}
}

func NewClientFromEnv() *Client {
	return NewClientFromKey(os.Getenv("FORWARDEMAIL_API_KEY"))
}

type Message struct {
	To      string `json:"to"`
	From    string `json:"from"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
	HTML    string `json:"html"`
	BCC     string `json:"bcc"`
}

func (c *Client) Send(ctx context.Context, m *Message) error {
	url := "https://api.forwardemail.net/v1/emails"

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(m); err != nil {
		return fmt.Errorf("error encoding message: %w", err)
	}

	req, _ := http.NewRequestWithContext(ctx, "POST", url, &buf)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Basic ZGFjZDBkYmQzNjJiZjg5YTlhMmRkOGQyOg==")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	_, err = io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	// fmt.Println(res)
	// fmt.Println(string(body))

	return nil
}
