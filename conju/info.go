package conju

import (
	"html/template"

	"google.golang.org/appengine/log"
)

func handleInfo(wr WrappedRequest) {
	eventName := "PSR2021"
	if wr.Event != nil {
		eventName = wr.Event.ShortName
	}
	var tpl = template.Must(template.ParseFiles("templates/main.html", "templates/"+eventName+"/info.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "info.html", wr.TemplateData); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}
