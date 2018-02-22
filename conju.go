package conju

import (
	"fmt"
)

func init() {
	AddSessionHandler("/test2", makeTemplateHandler("test.html", "test2.html")).Needs(EventGetter)
	AddSessionHandler("/test3", makeTemplateHandler("test.html", "test3.html"))
	AddSessionHandler("/importData", ImportData)
	AddSessionHandler("/increment", handleIncrement).Needs(EventGetter)
	AddSessionHandler("/resetData", clearAllData)

	AddSessionHandler("/listPeople", handleListPeople)
	AddSessionHandler("/updatePerson", handleUpdatePerson)
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
