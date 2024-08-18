package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"

	"cloud.google.com/go/datastore"
	"github.com/cshabsin/conju/tools/data/legacy"
)

func realMain(ctx context.Context) error {
	project := "useful-art-199822"
	datastoreClient, err := datastore.NewClient(ctx, project)
	if err != nil {
		return err
	}
	var events []*legacy.Event
	if _, err := datastoreClient.GetAll(ctx, datastore.NewQuery("Event"), &events); err != nil {
		return err
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].StartDate.Before(events[j].StartDate)
	})
	for _, e := range events {
		fmt.Println(e)
	}
	return nil
}

func main() {
	fmt.Println("yo")
	if err := realMain(context.Background()); err != nil {
		// In case of an auth error, try gcloud auth application-default login
		log.Fatal(err)
		os.Exit(1)
	}
}
