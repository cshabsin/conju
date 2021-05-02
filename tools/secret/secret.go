package main

import (
	"context"
	"fmt"
	"log"
	"os"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"

	"google.golang.org/api/iterator"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

func realMain(ctx context.Context) error {
	parent := "projects/useful-art-199822"
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return err
	}
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

func main() {
	if err := realMain(context.Background()); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
