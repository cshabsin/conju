package conju

import (
	"fmt"

	"google.golang.org/appengine/log"
)

func init() {
	AddSessionHandler("/test2", makeTemplateHandler("test.html", "test2.html"))
	AddSessionHandler("/test3", makeTemplateHandler("test.html", "test3.html"))
	AddSessionHandler("/importData", ImportData)
	AddSessionHandler("/increment", handleIncrement).Needs(EventGetter)
	AddSessionHandler("/cleanup", handleCleanup)
}

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
