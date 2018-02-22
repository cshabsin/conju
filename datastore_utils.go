package conju

import (
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

func clearAllData(wr WrappedRequest) {

	wr.Values["event"] = nil
	wr.SaveSession()

	//TODO: loop through all entity types

	q := datastore.NewQuery("CurrentEvent")
	for t := q.Run(wr.Context); ; {
		var ce CurrentEvent
		key, err := t.Next(&ce)
		if err == datastore.Done {
			break
		}
		if err != nil {
			log.Errorf(wr.Context, "%v", err)
			return
		}
		err = datastore.Delete(wr.Context, key)
		if err != nil {
			log.Errorf(wr.Context, "%v", err)
			return
		}
	}

	q = datastore.NewQuery("Person")
	for t := q.Run(wr.Context); ; {
		var p Person
		key, err := t.Next(&p)
		if err == datastore.Done {
			break
		}
		if err != nil {
			log.Errorf(wr.Context, "%v", err)
			return
		}
		err = datastore.Delete(wr.Context, key)
		if err != nil {
			log.Errorf(wr.Context, "%v", err)
			return
		}
	}

	return
}
