package conju

import (
	"fmt"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"html/template"
	"net/http"
	"time"
)

func init() {
	AddSessionHandler("/test2", makeTemplateHandler("test.html", "test2.html")).Needs(EventGetter)
	AddSessionHandler("/test3", makeTemplateHandler("test.html", "test3.html"))
	AddSessionHandler("/importData", ImportData)
	AddSessionHandler("/increment", handleIncrement).Needs(EventGetter)
	AddSessionHandler("/resetData", handleCleanup)

	AddSessionHandler("/listGuests", handleListGuests)}

func handleIncrement(wr WrappedRequest) {
	if wr.Values["n"] == nil {
		wr.Values["n"] = 0
	} else {
		wr.Values["n"] = wr.Values["n"].(int) + 1
	}
	wr.SaveSession()
	ev := wr.Event
	var event_name string
	if ev != nil {
		event_name = ev.Name
	}
	wr.ResponseWriter.Write([]byte(
		fmt.Sprintf("%s\n%d\n", event_name, wr.Values["n"].(int))))
}

func handleCleanup(wr WrappedRequest) {
	wr.Values["event"] = nil
	wr.SaveSession()
	// Deletes all "current event" objects
	err := cleanUp(wr.Context)
	if err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}


func handleListGuests(wr WrappedRequest) {

	ctx := appengine.NewContext(wr.Request)
	tic := time.Now()
	q := datastore.NewQuery("Person").Order("LastName").Order("FirstName")

	var allPeople []*Person
	if _, err := q.GetAll(ctx, &allPeople); err != nil {
		http.Error(wr.ResponseWriter, err.Error(), http.StatusInternalServerError)
		log.Errorf(ctx, "GetAll: %v", err)
		return
	}
	log.Infof(ctx, "Datastore lookup took %s", time.Since(tic).String())
	log.Infof(ctx, "Rendering %d people", len(allPeople))

	wr.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := struct {
		People           []*Person
	}{
		People:           allPeople,
	}

	var tpl = template.Must(template.ParseFiles("templates/test.html", "templates/listAllGuests.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "listAllGuests.html", data); err != nil {
		log.Errorf(ctx, "%v", err)
	}
}