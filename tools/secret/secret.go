package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iterator"
)

var project = flag.String("project", "useful-art-199822", "Google Cloud Project")
var listFlag = flag.Bool("list", false, "list secrets")
var accessFlag = flag.String("access", "", "secret id to access")
var createFlag = flag.String("create", "", "secret id to set")
var updateFlag = flag.String("update", "", "secret id to update")
var valueFlag = flag.String("value", "", "value to set in secret")

func realMain(ctx context.Context) error {
	// creds, err := credentials.DetectDefault(&credentials.DetectOptions{
	// 	Scopes: []string{"https://www.googleapis.com/auth/cloud-platform"},
	// })
	// if err != nil {
	// 	log.Fatalf("Error detecting credentials: %v", err)
	// }

	// // Now 'creds' contains information about the authenticated identity.
	// // You can potentially access details like the service account email or user ID.

	// // Example: Printing the credential type
	// log.Printf("Credential: %s", string(creds.JSON()))

	credentials, err := google.FindDefaultCredentials(ctx, compute.ComputeScope)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("projectID from credentials: ", credentials.ProjectID)

	parent := "projects/" + *project
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	if *listFlag {
		return listSecrets(ctx, client, parent)
	}
	if *createFlag != "" {
		return createSecret(ctx, client, parent, *createFlag, *valueFlag)
	}
	if *accessFlag != "" {
		if err := iamGrantAccess(ctx, client, os.Stdout, parent+"/secrets/"+*accessFlag, "user:cshabsin@gmail.com"); err != nil {
			return fmt.Errorf("failed to grant access: %w", err)
		}
		return accessSecret(ctx, client, parent, *accessFlag)
	}
	return errors.New("no action specified")
}

func iamGrantAccess(ctx context.Context, client *secretmanager.Client, w io.Writer, name, member string) error {
	// name := "projects/my-project/secrets/my-secret"
	// member := "user:foo@example.com"

	// Get the current IAM policy.
	handle := client.IAM(name)
	policy, err := handle.Policy(ctx)
	if err != nil {
		return fmt.Errorf("failed to get policy: %w", err)
	}

	// Grant the member access permissions.
	policy.Add(member, "roles/secretmanager.secretAccessor")
	if err = handle.SetPolicy(ctx, policy); err != nil {
		return fmt.Errorf("failed to save policy: %w", err)
	}

	fmt.Fprintf(w, "Updated IAM policy for %s\n", name)
	return nil
}

func listSecrets(ctx context.Context, client *secretmanager.Client, parent string) error {
	req := &secretmanagerpb.ListSecretsRequest{
		Parent: parent,
	}
	it := client.ListSecrets(ctx, req)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		fmt.Printf("Found secret %q\n", resp.Name)
	}
	return nil
}

func createSecret(ctx context.Context, client *secretmanager.Client, parent, secretID, value string) error {
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
	resp, err := client.CreateSecret(ctx, req)
	if err != nil {
		return err
	}
	fmt.Printf("Created secret %q\n", resp.Name)
	addSecretVersionReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: resp.Name,
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(value),
		},
	}
	_, err = client.AddSecretVersion(ctx, addSecretVersionReq)
	return err
}

func accessSecret(ctx context.Context, client *secretmanager.Client, parent, secretID string) error {
	fqName := "projects/" + parent + "/secrets/" + secretID + "/versions/latest"
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fqName,
	}
	response, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get secret: %v", err)
	}
	fmt.Printf("Retrieved secret %q: %q\n", response.GetName(), string(response.GetPayload().GetData()))
	return nil
}

func main() {
	flag.Parse()
	if err := realMain(context.Background()); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
