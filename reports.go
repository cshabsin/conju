package conju

import (
	//	"context"
	"html/template"
	//	"net/http"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

func handleReports(wr WrappedRequest) {
	var tpl = template.Must(template.ParseFiles("templates/main.html", "templates/reports.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "reports.html", wr.TemplateData); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}

func handleRsvpReport(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	currentEventKeyEncoded := wr.Values["EventKey"].(string)
	currentEventKey, _ := datastore.DecodeKey(currentEventKeyEncoded)

	var invitations []*Invitation
	q := datastore.NewQuery("Invitation").Filter("Event =", currentEventKey)
	_, err := q.GetAll(ctx, &invitations)
	if err != nil {
		log.Errorf(ctx, "fetching invitations: %v", err)
	}

	allRsvpMap := make(map[RsvpStatus][][]Person)
	var allNoRsvp [][]Person

	for _, invitation := range invitations {

		rsvpMap, noRsvp := invitation.ClusterByRsvp(ctx)

		for r, p := range rsvpMap {
			listOfLists := allRsvpMap[r]
			if listOfLists == nil {
				listOfLists = make([][]Person, 0)
			}
			listOfLists = append(listOfLists, p)
			allRsvpMap[r] = listOfLists
		}
		if len(noRsvp) > 0 {
			allNoRsvp = append(allNoRsvp, noRsvp)
		}
	}

	statusOrder := []RsvpStatus{ThuFriSat, FriSat, Maybe, No}

	tpl := template.Must(template.New("").ParseFiles("templates/main.html", "templates/rsvpReport.html"))
	data := wr.MakeTemplateData(map[string]interface{}{
		"RsvpMap":         allRsvpMap,
		"NoRsvp":          allNoRsvp,
		"StatusOrder":     statusOrder,
		"AllRsvpStatuses": GetAllRsvpStatuses(),
	})
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "rsvpReport.html", data); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}

func handleActivitiesReport(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	currentEventKeyEncoded := wr.Values["EventKey"].(string)
	currentEventKey, _ := datastore.DecodeKey(currentEventKeyEncoded)

	var invitations []*Invitation
	q := datastore.NewQuery("Invitation").Filter("Event =", currentEventKey)
	_, err := q.GetAll(ctx, &invitations)
	if err != nil {
		log.Errorf(ctx, "fetching invitations: %v", err)
	}

	allRsvpStatuses := GetAllRsvpStatuses()

	var activities = make([]*Activity, len(wr.Event.Activities))
	err = datastore.GetMulti(ctx, wr.Event.Activities, activities)
	if err != nil {
		log.Errorf(ctx, "fetching activities: %v", err)
	}

	keysToActivities := make(map[datastore.Key]Activity)
	activityKeys := make([]datastore.Key, len(wr.Event.Activities))
	for i, key := range wr.Event.Activities {
		keysToActivities[*key] = *activities[i]
		activityKeys[i] = *key
	}

	type ActivityResponse struct {
		NoResponses         []datastore.Key
		MaybeResponses      []datastore.Key
		DefinitelyResponses []datastore.Key
		Leaders             []datastore.Key
	}

	activityResponseMap := make(map[datastore.Key]*ActivityResponse)
	for _, activityKey := range wr.Event.Activities {
		activityResponseMap[*activityKey] = &ActivityResponse{}
	}

	var allPeopleToLookUp []*datastore.Key
	for _, invitation := range invitations {
		if invitation.RsvpMap == nil {
			continue
		}

		if invitation.ActivityMap == nil {
			continue
		}

		personKeySet := make(map[datastore.Key]bool)
		for k, v := range invitation.RsvpMap {
			if allRsvpStatuses[v].Attending {
				personKeySet[*k] = true
				allPeopleToLookUp = append(allPeopleToLookUp, k)
			}
		}

		for k, v := range invitation.ActivityMap {

			if _, present := personKeySet[*k]; present {
				for ak, preference := range v {
					response := activityResponseMap[*ak]
					switch preference {
					case ActivityNo:
						response.NoResponses = append(response.NoResponses, *k)
					case ActivityMaybe:
						response.MaybeResponses = append(response.MaybeResponses, *k)
					case ActivityDefinitely:
						response.DefinitelyResponses = append(response.DefinitelyResponses, *k)
					}
				}
			}
		}
	}

	var people = make([]*Person, len(allPeopleToLookUp))
	err = datastore.GetMulti(ctx, allPeopleToLookUp, people)
	if err != nil {
		log.Errorf(ctx, "fetching people: %v", err)
	}

	personMap := make(map[datastore.Key]string)
	for i, person := range people {
		personMap[*allPeopleToLookUp[i]] = person.FullNameWithAge(wr.Event.StartDate)
	}

	tpl := template.Must(template.New("").ParseFiles("templates/main.html", "templates/activitiesReport.html"))
	data := wr.MakeTemplateData(map[string]interface{}{
		"ActivityKeys":        activityKeys,
		"KeysToActivities":    keysToActivities,
		"ActivityResponseMap": activityResponseMap,
		"PersonMap":           personMap,
	})
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "activitiesReport.html", data); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}

}
