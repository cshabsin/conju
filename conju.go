package conju

import "net/http"

func init() {
	http.HandleFunc("/t", makeTemplateHandler("test.html", "test3.html"))
	http.HandleFunc("/", makeTemplateHandler("test.html", "test2.html"))
}
