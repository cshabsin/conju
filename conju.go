package conju

import "net/http"

func init() {
	http.HandleFunc("/test2", makeTemplateHandler("test.html", "test2.html"))
	http.HandleFunc("/test3", makeTemplateHandler("test.html", "test3.html"))
}
