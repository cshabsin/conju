package main

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/datastore"
	"google.golang.org/appengine/v2"

	"github.com/cshabsin/conju/conju"
	"github.com/cshabsin/conju/conju/mailer/forwardemail"
	"github.com/cshabsin/conju/conju/secretmanager"
	"github.com/cshabsin/conju/view/poll"
)

func main() {
	ctx := context.Background()
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Fatal("GOOGLE_CLOUD_PROJECT environment variable is not set")
	}
	log.Printf("Using Google Cloud Project: %s", projectID)
	// Initialize Firestore client
	datastoreClient, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer datastoreClient.Close()

	secretmanagerClient, err := secretmanager.NewClient(ctx)
	if err != nil {
		log.Fatalf("Could not create secret manager client: %v", err)
	}

	mailClient, err := forwardemail.NewClientFromSecretManager(ctx, secretmanagerClient)
	if err != nil {
		log.Fatalf("Could not create mail client: %v", err)
	}

	s := conju.Sessionizer{
		DatastoreClient:     datastoreClient,
		SecretmanagerClient: secretmanagerClient,
		MailClient:          mailClient,
	}

	conju.Register(s)
	poll.Register(s)

	appengine.Main()
}
