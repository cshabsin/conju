package conju

import (
	"net/http"

	"google.golang.org/appengine"
)

func init() {
	AddSessionHandler("/test2", makeTemplateHandler("test.html", "test2.html"))
	AddSessionHandler("/test3", makeTemplateHandler("test.html", "test3.html"))
	AddSessionHandler("/create", handleCreate)
	AddSessionHandler("/importData", ImportData)

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
