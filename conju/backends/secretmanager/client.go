package secretmanager

import (
	"context"
	"fmt"

	"cloud.google.com/go/compute/metadata"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

type Client struct {
	client *secretmanager.Client
}

func NewClient(ctx context.Context) (*Client, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Client{client: client}, nil
}

func FromContext(ctx context.Context) *Client {
	return ctx.Value(key).(*Client)
}

func WrapContext(ctx context.Context, client *Client) context.Context {
	return context.WithValue(ctx, key, client)
}

var key = &struct{}{}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) Create(ctx context.Context, parent, secretID, value string) error {
	if secretID == "" {
		return fmt.Errorf("no secret id specified")
	}
	if value == "" {
		return fmt.Errorf("no value specified for secret %q", secretID)
	}
	req := &secretmanagerpb.CreateSecretRequest{
		Parent:   parent,
		SecretId: secretID,
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		},
	}
	resp, err := c.client.CreateSecret(ctx, req)
	if err != nil {
		return err
	}
	addSecretVersionReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: resp.Name,
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(value),
		},
	}
	_, err = c.client.AddSecretVersion(ctx, addSecretVersionReq)
	return err
}

func (c *Client) Get(ctx context.Context, secretID string) (string, error) {
	var projectID string
	if metadata.OnGCE() {
		var err error
		projectID, err = metadata.ProjectIDWithContext(ctx)
		if err != nil {
			return "", err
		}
	}
	fqName := "projects/" + projectID + "/secrets/" + secretID + "/versions/latest"
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fqName,
	}
	response, err := c.client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to get secret: %v", err)
	}
	return string(response.GetPayload().GetData()), nil
}
