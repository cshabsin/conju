package conju

import (
	"fmt"
	"net/http"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

func ClearAllData(wr WrappedRequest) {
	fmt.Fprintf(wr.ResponseWriter, "Disabled for now.")
	// wr.Values["event"] = nil
	// wr.SaveSession()

	// entityNames := []string{"Activity", "Event", "CurrentEvent", "Person", "Invitation", "LoginCode", "Venue", "Building", "Room"}

	// for _, entityName := range entityNames {
	// 	wr.ResponseWriter.Write([]byte(fmt.Sprintf("Clearing: %s\n", entityName)))
	// 	q := datastore.NewQuery(entityName).KeysOnly()

	// 	keys, err := q.GetAll(wr.Context, nil)
	// 	if err != nil {
	// 		log.Errorf(wr.Context, "%v", err)
	// 		return
	// 	}

	// 	wr.ResponseWriter.Write([]byte(
	// 		fmt.Sprintf("	%d %s to delete\n", len(keys), entityName)))

	// 	if err != nil {
	// 		log.Errorf(wr.Context, "%v", err)
	// 		return
	// 	}

	// 	err = datastore.DeleteMulti(wr.Context, keys)
	// 	if err != nil {
	// 		log.Errorf(wr.Context, "%v", err)
	// 		return
	// 	}
	// }
}

func RepairData(wr WrappedRequest) {
	q := datastore.NewQuery("Person")
	var people []Person
	personKeys, err := q.GetAll(wr.Context, &people)
	if err != nil {
		log.Errorf(wr.Context, "RepairData personQuery: %v", err)
		http.Error(wr.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return
	}
	for i := range personKeys {
		if people[i].LoginCode == "" {
			people[i].LoginCode = randomLoginCodeString()
			_, err = datastore.Put(wr.Context, personKeys[i], &people[i])
			if err != nil {
				log.Errorf(wr.Context, "RepairData put(%s): %v", people[i].Email, err)
				http.Error(wr.ResponseWriter, fmt.Sprintf("put(%s): %v", people[i].Email, err), http.StatusInternalServerError)
				return
			}
		}
	}
	fmt.Fprintf(wr.ResponseWriter, "Done.")
}
