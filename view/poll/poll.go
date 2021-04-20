package poll

import (
	"html/template"
	"net/http"

	"github.com/cshabsin/conju/conju"
	"github.com/cshabsin/conju/model/poll"
	"google.golang.org/appengine/log"
)

func HandlePoll(wr *conju.WrappedRequest) {
	ctx := wr.Context
	if wr.Invitation == nil {
		http.Redirect(wr.ResponseWriter, wr.Request, "/login", http.StatusSeeOther)
	}
	key, poll, err := poll.GetAnswer(ctx, wr.InvitationKey)
	if err != nil {
		log.Errorf(ctx, "error reading answer: %v", err)
		http.Error(wr.ResponseWriter, "error reading answer", http.StatusInternalServerError)
		return
	}
	data := wr.TemplateData
	if key != nil {
		data["pollEncodedKey"] = key.Encode()
		data["rating"] = poll.Rating
	}
	tpl := template.Must(template.New("").ParseFiles("templates/main.html", "templates/poll.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "poll.html", data); err != nil {
		log.Errorf(wr.Context, "error executing poll template %v", err)
		http.Error(wr.ResponseWriter, "error executing poll template", http.StatusInternalServerError)
	}
}
