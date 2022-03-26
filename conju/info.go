package conju

import (
	"html/template"
	"log"
)

func handleInfo(wr WrappedRequest) {
	eventName := "PSR2022"
	if wr.Event != nil {
		eventName = wr.Event.ShortName
	}
	tpl, err := template.ParseFiles("templates/main.html", "templates/"+eventName+"/info.html")
	if err != nil {
		log.Println("info ParseFiles", err)
	}
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "info.html", wr.TemplateData); err != nil {
		log.Println("info ExecuteTemplate", err)
	}
}
