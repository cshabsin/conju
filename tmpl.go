package conju

import (
	"html/template"

	"google.golang.org/appengine/log"
)

// Convert template filenames to template paths (add templates/ prefix).
func templatePaths(fns []string) []string {
	paths := make([]string, len(fns))
	for i, fn := range fns {
		paths[i] = "templates/" + fn
	}
	return paths
}

// makeTmplateHandler creates an HTTP handler that renders a template with
// standard data providers (yet to be determined). The function takes one or
// more template filenames to parse, and returns a provider that executes
// the last template in the list.
func makeTemplateHandler(ts ...string) func(WrappedRequest) {
	tpl := template.Must(template.ParseFiles(templatePaths(ts)...))
	return func(wr WrappedRequest) {
		if err := tpl.ExecuteTemplate(wr.ResponseWriter, ts[len(ts)-1], nil); err != nil {
			log.Errorf(wr.Context, "%v", err)
		}
	}
}
