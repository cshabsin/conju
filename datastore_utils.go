package conju

import (
	"fmt"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

func ClearAllData(wr WrappedRequest) {

	wr.Values["event"] = nil
	wr.SaveSession()

	entityNames := []string{"Activity", "Event", "Person", "Invitation"}

	for _, entityName := range entityNames {
		wr.ResponseWriter.Write([]byte(fmt.Sprintf("Clearing: %s\n", entityName)))
		q := datastore.NewQuery(entityName).KeysOnly()

		keys, err := q.GetAll(wr.Context, nil)
		if err != nil {
			log.Errorf(wr.Context, "%v", err)
			return
		}

		wr.ResponseWriter.Write([]byte(
			fmt.Sprintf("	%d %s to delete\n", len(keys), entityName)))

		if err != nil {
			log.Errorf(wr.Context, "%v", err)
			return
		}

		err = datastore.DeleteMulti(wr.Context, keys)
		if err != nil {
			log.Errorf(wr.Context, "%v", err)
			return
		}
	}

	return
}
