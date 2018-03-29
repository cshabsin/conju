package conju

import (
	"bytes"
	"context"
	"html/template"

	"google.golang.org/appengine/log"
	"google.golang.org/appengine/mail"
)

type MailHeaderInfo struct {
	To      []string
	Cc      []string
	Bcc     []string
	Sender  string
	Subject string
}

func sendMail(ctx context.Context, templatePrefix string, data interface{}, functions template.FuncMap, headerData MailHeaderInfo) {

	file := "templates/email/" + templatePrefix + ".html"
	tpl := template.Must(template.New("").Funcs(functions).ParseFiles(file))
	var text bytes.Buffer
	var html bytes.Buffer
	if err := tpl.ExecuteTemplate(&text, templatePrefix+"_text", data); err != nil {
		log.Errorf(ctx, "%v", err)
	}
	if err := tpl.ExecuteTemplate(&html, templatePrefix+"_html", data); err != nil {
		log.Errorf(ctx, "%v", err)
	}

	msg := &mail.Message{
		Sender:   headerData.Sender,
		To:       headerData.To,
		Subject:  headerData.Subject,
		Body:     text.String(),
		HTMLBody: html.String(),
	}
	log.Infof(ctx, text.String())
	if err := mail.Send(ctx, msg); err != nil {
		log.Errorf(ctx, "Couldn't send email: %v", err)
	}

}
