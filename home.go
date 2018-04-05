package conju

import (
	"html/template"
	"time"

	"google.golang.org/appengine/log"
)

func handleHomePage(wr WrappedRequest) {
	functionMap := template.FuncMap{
		"ShortDate":    shortDate,
		"MaybeDayOnly": maybeDayOnly,
	}
	tpl := template.Must(template.New("").
		Funcs(functionMap).
		ParseFiles("templates/main.html", "templates/index.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "index.html", wr.TemplateData); err != nil {
		log.Errorf(wr.Context, "%v", err)
		return
	}
}

func shortDate(t time.Time) string {
	return t.Format("Jan 2")
}

func maybeDayOnly(t1 time.Time, t2 time.Time) string {
	if t1.Month() == t2.Month() {
		return t1.Format("2")
	}
	return shortDate(t1)
}
