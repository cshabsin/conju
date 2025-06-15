package conju

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/datastore"

	"github.com/cshabsin/conju/conju/dsclient"
	"github.com/cshabsin/conju/conju/login"
	"github.com/cshabsin/conju/model/person"
)

func ClearAllData(ctx context.Context, wr WrappedRequest, entityNames []string) {
	fmt.Fprintf(wr.ResponseWriter, "Disabled for now.\n")
	wr.Values["event"] = nil
	wr.SaveSession()

	//entityNames := []string{"Activity", "Event", "CurrentEvent", "Person", "Invitation", "LoginCode", "Venue", "Building", "Room"}

	for _, entityName := range entityNames {
		wr.ResponseWriter.Write([]byte(fmt.Sprintf("Clearing: %s\n", entityName)))
		q := datastore.NewQuery(entityName).KeysOnly()

		keys, err := dsclient.FromContext(ctx).GetAll(ctx, q, nil)
		if err != nil {
			log.Println("ClearAllData GetAll:", err)
			return
		}

		_, err = wr.ResponseWriter.Write([]byte(
			fmt.Sprintf("	%d %s to delete\n", len(keys), entityName)))

		if err != nil {
			log.Println("ClearAllData Write:", err)
			return
		}

		err = dsclient.FromContext(ctx).DeleteMulti(ctx, keys)
		if err != nil {
			log.Println("ClearAllData DeleteMulti:", err)
			return
		}
	}
}

func RepairData(ctx context.Context, wr WrappedRequest) {
	q := datastore.NewQuery("Person")
	var people []person.Person
	personKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &people)
	if err != nil {
		log.Printf("RepairData personQuery: %v", err)
		http.Error(wr.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return
	}
	for i := range personKeys {
		if people[i].LoginCode == "" {
			people[i].LoginCode = login.RandomLoginCodeString()
			_, err = dsclient.FromContext(ctx).Put(ctx, personKeys[i], &people[i])
			if err != nil {
				log.Printf("RepairData put(%s): %v", people[i].Email, err)
				http.Error(wr.ResponseWriter, fmt.Sprintf("put(%s): %v", people[i].Email, err), http.StatusInternalServerError)
				return
			}
		}
	}
	fmt.Fprintf(wr.ResponseWriter, "Done.")
}
