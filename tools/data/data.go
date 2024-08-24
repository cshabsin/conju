package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/datastore"
	"github.com/cshabsin/conju/tools/data/keyutil"
	"github.com/cshabsin/conju/tools/data/legacy"
)

func realMain(ctx context.Context) error {
	project := "useful-art-199822"
	datastoreClient, err := datastore.NewClient(ctx, project)
	if err != nil {
		return err
	}
	// events, err := db.GetAll[*legacy.Event](ctx, datastoreClient, "Event")
	var events []*legacy.Event
	keys, err := datastoreClient.GetAll(ctx, datastore.NewQuery("Event"), &events)
	if err != nil {
		return err
	}
	eventMap := keyutil.ToMap(keys, events)
	for _, e := range eventMap {
		fmt.Println(e)
	}
	return nil
}

func main() {
	if err := realMain(context.Background()); err != nil {
		// In case of an auth error, try gcloud auth application-default login
		log.Fatal(err)
		os.Exit(1)
	}
}
