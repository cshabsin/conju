package conju

import (
	"html/template"
	"net/http"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

func init() {
	http.HandleFunc("/", handler)
}

var tpl = template.Must(template.ParseGlob("templates/*.html"))

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	if err := tpl.ExecuteTemplate(w, "test2.html", nil); err != nil {
		log.Errorf(ctx, "%v", err)
	}
}
