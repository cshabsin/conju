package conju

import (
	"html/template"

	"google.golang.org/appengine/log"
)

func handleFaq(wr WrappedRequest) {
	eventName := "PSR2021"
	if wr.Event != nil {
		eventName = wr.Event.ShortName
	}
	var tpl = template.Must(template.ParseFiles("templates/main.html", "templates/"+eventName+"/faq.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "faq.html", wr.TemplateData); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}
