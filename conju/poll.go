package conju

import (
	"html/template"
	"log"
	"net/http"

	"github.com/cshabsin/conju/model/poll"
	"github.com/gin-gonic/gin"
)

func HandlePoll(c *gin.Context) {
	wr, ok := c.MustGet("wrappedRequest").(*WrappedRequest)
	if !ok {
		log.Printf("could not get wrapped request from context")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	ctx := c.Request.Context()

	if wr.Invitation == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	key, poll, err := poll.GetAnswer(ctx, wr.InvitationKey)
	if err != nil {
		log.Printf("error reading answer: %v", err)
		c.String(http.StatusInternalServerError, "error reading answer")
		return
	}
	data := wr.TemplateData
	if key != nil {
		data["pollEncodedKey"] = key.Encode()
		data["rating"] = poll.Rating
	} else {
		data["pollEncodedKey"] = "not found"
		data["rating"] = "unset"
	}
	tpl := template.Must(template.New("").ParseFiles("templates/main.html", "templates/poll.html"))
	if err := tpl.ExecuteTemplate(c.Writer, "poll.html", data); err != nil {
		log.Printf("error executing poll template %v", err)
		c.String(http.StatusInternalServerError, "error executing poll template")
	}
}
