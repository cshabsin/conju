package conju

import (
	"context"
	"fmt"
	"html/template"
	"log"
)

func handleInfo(ctx context.Context, wr *WrappedRequest) {
	wr.ResponseWriter.Header().Set("Content-Type", "text/html")
	eventName := "PSR2022"
	if wr.Event != nil {
		eventName = wr.Event.ShortName
	}
	tpl, err := template.ParseFiles("templates/main.html", "templates/"+eventName+"/info.html")
	if err != nil {
		log.Println("info ParseFiles", err)
		fmt.Fprintf(wr.ResponseWriter, "Error: %v<p>", err)
		return
	}
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "info.html", wr.TemplateData); err != nil {
		log.Println("info ExecuteTemplate", err)
	}
}
