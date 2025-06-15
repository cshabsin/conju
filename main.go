package main

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/datastore"
	"google.golang.org/appengine/v2"

	"github.com/cshabsin/conju/conju"
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

	conju.Register(datastoreClient)
	// poll.Register(datastoreClient)

	appengine.Main()
}
