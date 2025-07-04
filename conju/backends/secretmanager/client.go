package secretmanager

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/compute/metadata"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"google.golang.org/api/iterator"
)

type Client struct {
	projectID string
	client    *secretmanager.Client
}

func NewClient(ctx context.Context) (*Client, error) {
	var projectID string
	if metadata.OnGCE() {
		var err error
		projectID, err = metadata.ProjectIDWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Client{projectID: projectID, client: client}, nil
}

func FromContext(ctx context.Context) (*Client, error) {
	c, ok := ctx.Value(contextKey{}).(*Client)
	if !ok {
		return nil, fmt.Errorf("no secret manager client in context")
	}
	return c, nil
}

func WrapContext(ctx context.Context, client *Client) context.Context {
	return context.WithValue(ctx, contextKey{}, client)
}

type contextKey struct{}

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

func (c *Client) Get(ctx context.Context, secretID string) ([]byte, error) {
	fqName := "projects/" + c.projectID + "/secrets/" + secretID + "/versions/latest"
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fqName,
	}
	response, err := c.client.AccessSecretVersion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %v", err)
	}
	return response.GetPayload().GetData(), nil
}

// Use gracePeriodDays=0 to get all active versions.
func (c *Client) GetVersionsWithinGracePeriod(ctx context.Context, secretID string, gracePeriodDays int) ([][]byte, error) {
	parentName := "projects/" + c.projectID + "/secrets/" + secretID
	req := &secretmanagerpb.ListSecretVersionsRequest{
		Parent: parentName,
	}
	var keys [][]byte
	cutoffTime := time.Now().AddDate(0, 0, -gracePeriodDays)
	it := c.client.ListSecretVersions(ctx, req)
	for {
		version, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if version.State != secretmanagerpb.SecretVersion_ENABLED {
			continue
		}
		if gracePeriodDays != 0 && version.CreateTime.AsTime().Before(cutoffTime) {
			continue
		}
		result, err := c.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
			Name: version.Name,
		})
		if err != nil {
			return nil, err
		}
		log.Printf("including secret %v created %v", version.Name, version.CreateTime.AsTime())
		keys = append(keys, result.Payload.Data)
	}
	return keys, nil
}
