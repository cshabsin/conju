package conju

import (
	"net/http"

	"google.golang.org/appengine"
)

func init() {
	AddSessionHandler("/test2", makeTemplateHandler("test.html", "test2.html"))
	AddSessionHandler("/test3", makeTemplateHandler("test.html", "test3.html"))
	http.HandleFunc("/create", handleCreate)
	http.HandleFunc("/importData", ImportData)

}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	err := CreateOneOffEvent(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}
