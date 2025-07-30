package conju

import (
	"html/template"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func handleInfo(c *gin.Context) {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	c.Header("Content-Type", "text/html")
	eventName := "PSR2022"
	if wr.Event != nil {
		eventName = wr.Event.ShortName
	}
	tpl, err := template.ParseFiles("templates/main.html", "templates/"+eventName+"/info.html")
	if err != nil {
		log.Println("info ParseFiles", err)
		c.String(http.StatusInternalServerError, "Error: %v<p>", err)
		return
	}
	if err := tpl.ExecuteTemplate(c.Writer, "info.html", wr.TemplateData); err != nil {
		log.Println("info ExecuteTemplate", err)
	}
}