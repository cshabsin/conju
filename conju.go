package conju

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

func init() {
	http.HandleFunc("/test2", makeTemplateHandler("test.html", "test2.html"))
	http.HandleFunc("/test3", makeTemplateHandler("test.html", "test3.html"))
	http.HandleFunc("/create", handleCreate)
	http.HandleFunc("/testCollectiveName", testCollectiveName)
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	err := CreateOneOffEvent(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	CreatePerson(ctx, "Christopher", "Shabsin")
	CreatePerson(ctx, "Dana", "Scott")
	CreatePerson(ctx, "Lydia", "Shabsin")

	CreatePerson(ctx, "Matthew", "Carter")
	CreatePerson(ctx, "Sarah", "Carter")
	CreatePerson(ctx, "Geneva", "Carter")
	CreatePerson(ctx, "Owen", "Carter")
	CreatePerson(ctx, "Beatrix", "Carter")

	CreatePerson(ctx, "Zachary", "Ananian")
	CreatePerson(ctx, "Zachary", "St. Lawrence")

	http.Redirect(w, r, "/", http.StatusFound)
}

type TestData struct {
	label         string
	filteredField string
	fieldValue    string
}

func testCollectiveName(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	//qSinglePerson := datastore.NewQuery("Person").Filter("LastName =", "Scott")
	//qCoupleDifferentLastNames := datastore.NewQuery("Person").Filter("FirstName =", "Zachary")
	//qCoupleSameLastName := datastore.NewQuery("Person").Filter("LastName = ", "Shabsin")
	//qManyDifferentLastNames := datastore.NewQuery("Person")
	//qManySameLastName :=  datastore.NewQuery("Person").Filter("LastName = ", "Carter")

	toTest := []TestData{
		TestData{"Single Person", "LastName =", "Scott"},
		TestData{"Couple, Different Last Names", "FirstName = ", "Zachary"},
		TestData{"Couple, Same Last Name", "LastName =", "Shabsin"},
		//TestData{">2 People, Different Last Names", "", ""},
		TestData{">2 People, Same Last Name", "LastName = ", "Carter"},
	}

	b := new(bytes.Buffer)
	for t := 0; t < len(toTest); t++ {
		test := toTest[t]
		//query := datastore.NewQuery("Person")
		//if test.filteredField != "" {
		query := datastore.NewQuery("Person").Filter(test.filteredField, test.fieldValue)
		//}

		fmt.Fprintln(b, "-----  "+test.label+"  -----")
		result := query.Run(ctx)
		resultSlice := []Person{}
		for p := result; ; {
			var x Person
			_, err := p.Next(&x)
			if err == datastore.Done {
				break
			}
			if err != nil {
				//serveError(ctx, w, err)
				return
			}
			fmt.Fprintf(b, "%v\n", x.FullNameWithFormality(Full))
			resultSlice = append(resultSlice, x)
		}
		fmt.Fprintf(b, "\n -->  %v\n\n", CollectiveAddress(resultSlice, Informal))
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.Copy(w, b)
}
