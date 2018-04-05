package conju

import (
	"google.golang.org/appengine/log"
	"html/template"
)

func handleHomePage(wr WrappedRequest) {
	tpl := template.Must(template.New("").ParseFiles("templates/main.html", "templates/index.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "index.html", wr.TemplateData); err != nil {
		log.Errorf(wr.Context, "%v", err)
		return
	}
}
