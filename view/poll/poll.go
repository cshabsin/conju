package poll

import (
	"context"
	"html/template"
	"log"
	"net/http"

	"github.com/cshabsin/conju/conju"
	"github.com/cshabsin/conju/model/poll"
)

func Register(s conju.Sessionizer) {
	s.AddSessionHandler("/poll", HandlePoll).Needs(conju.InvitationGetter)
}

func HandlePoll(ctx context.Context, wr conju.WrappedRequest) {
	if wr.Invitation == nil {
		http.Redirect(wr.ResponseWriter, wr.Request, "/login", http.StatusSeeOther)
	}
	key, poll, err := poll.GetAnswer(ctx, wr.InvitationKey)
	if err != nil {
		log.Printf("error reading answer: %v", err)
		http.Error(wr.ResponseWriter, "error reading answer", http.StatusInternalServerError)
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
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "poll.html", data); err != nil {
		log.Printf("error executing poll template %v", err)
		http.Error(wr.ResponseWriter, "error executing poll template", http.StatusInternalServerError)
	}
}
