package conju

import (
	"net/http"
	"strconv"

	"google.golang.org/appengine"
)

func init() {
	AddSessionHandler("/test2", makeTemplateHandler("test.html", "test2.html"))
	AddSessionHandler("/test3", makeTemplateHandler("test.html", "test3.html"))
	AddSessionHandler("/create", handleCreate)
	AddSessionHandler("/importData", ImportData)
	AddSessionHandler("/increment", handleIncrement)
}

func handleCreate(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	err := CreateOneOffEvent(ctx)
	if err != nil {
		http.Error(wr.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(wr.ResponseWriter, wr.Request, "/", http.StatusFound)
}

func handleIncrement(wr WrappedRequest) {
	if wr.Values["n"] == nil {
		wr.Values["n"] = 0
	} else {
		wr.Values["n"] = wr.Values["n"].(int) + 1
	}
	wr.SaveSession()
	wr.ResponseWriter.Write([]byte(strconv.Itoa(wr.Values["n"].(int))))
}
