package conju

// TODO: move to "package models"?

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

type CurrentEvent struct {
	Key *datastore.Key
}

// TODO: add object that's a map of string names to values and attach one to every event
type Event struct {
	EventId               int // this can get deleted after all the data is imported
	Venue                 *datastore.Key
	Name                  string
	ShortName             string
	StartDate             time.Time
	EndDate               time.Time
	RsvpStatuses          []RsvpStatus
	Rooms                 []*datastore.Key
	Activities            []*datastore.Key
	InvitationClosingText string
	Current               bool
}

// Sets up Event in the WrappedRequest.
func EventGetter(wr *WrappedRequest) error {
	if wr.hasRunEventGetter {
		return nil // Only retrieve once.
	}
	wr.hasRunEventGetter = true
	key, err := wr.RetrieveKeyFromSession("EventKey")
	if err != nil {
		return err
	}
	var e Event
	err = datastore.Get(wr.Context, key, &e)
	if err == nil {
		// We have retrieved the event successfully.
		wr.Event = &e
		wr.EventKey = key
		wr.TemplateData["CurrentEvent"] = e
		return nil
	}

	var keys []*datastore.Key
	var events []*Event
	q := datastore.NewQuery("Event").Filter("Current =", true)
	keys, err = q.GetAll(wr.Context, &events)
	if err != nil {
		log.Errorf(wr.Context, "Error querying for current event: %v", err)
		return nil
	}
	if len(keys) == 0 {
		log.Errorf(wr.Context, "Found no current event")
		return nil
	}
	if len(keys) > 1 {
		log.Errorf(wr.Context, "Found more than one current event (%d)", len(keys))
		return nil
	}
	wr.Event = events[0]
	key = keys[0]

	wr.TemplateData["CurrentEvent"] = wr.Event
	wr.EventKey = key
	wr.SetSessionValue("EventKey", key.Encode())
	wr.SaveSession()

	return nil
}

func handleEvents(wr WrappedRequest) {

	ctx := appengine.NewContext(wr.Request)
	tic := time.Now()
	q := datastore.NewQuery("Event").Order("-StartDate")

	var allEvents []*Event
	_, err := q.GetAll(ctx, &allEvents)
	if err != nil {
		http.Error(wr.ResponseWriter, err.Error(), http.StatusInternalServerError)
		log.Errorf(ctx, "GetAll: %v", err)
		return
	}
	log.Infof(ctx, "Datastore lookup took %s", time.Since(tic).String())
	log.Infof(ctx, "Rendering %d events", len(allEvents))

	q = datastore.NewQuery("Venue")
	var allVenues []*Venue
	venueKeys, _ := q.GetAll(ctx, &allVenues)

	venueMap := make(map[datastore.Key]Venue)
	venueEncodedKeyMap := make(map[datastore.Key]string)
	for i := 0; i < len(allVenues); i++ {
		venueMap[*venueKeys[i]] = *allVenues[i]
		venueEncodedKeyMap[*venueKeys[i]] = (*venueKeys[i]).Encode()
	}

	// fetch this with an ajax call eventually
	var buildings []Building
	var buildingPtrs []*Building
	var buildingInts []int64
	var rooms []*Room
	buildingRoomMap := make(map[int64][]Room)
	buildingKeyMap := make(map[int64]Building)
	if len(allVenues) == 1 {
		q := datastore.NewQuery("Building").Ancestor(venueKeys[0]).Order("Name")
		buildingKeys, _ := q.GetAll(ctx, &buildingPtrs)

		for i := 0; i < len(buildingPtrs); i++ {
			buildings = append(buildings, *(buildingPtrs[i]))
			buildingInts = append(buildingInts, buildingKeys[i].IntID())
			var roomList []Room
			buildingRoomMap[(buildingInts[i])] = roomList
			buildingKeyMap[buildingKeys[i].IntID()] = buildings[i]

		}

		q = datastore.NewQuery("Room").Order("RoomNumber").Order("Partition")
		_, _ = q.GetAll(ctx, &rooms)

		for j := 0; j < len(rooms); j++ {
			//building := buildingKeyMap[rooms[j].Building.IntID()]
			//log.Infof(ctx, "Found room %d for building %s", rooms[j].RoomNumber, building.Name)
			roomList := buildingRoomMap[rooms[j].Building.IntID()]
			roomList = append(roomList, *(rooms[j]))
			buildingRoomMap[rooms[j].Building.IntID()] = roomList
			//log.Infof(ctx, "room list size: %d", len(roomList))
		}
	}

	var activities []Activity
	q = datastore.NewQuery("Activity").Order("Keyword")
	activityKeys, _ := q.GetAll(ctx, &activities)

	var activitiesWithKeys []ActivityWithKey
	for i, activityKey := range activityKeys {
		encodedKey := activityKey.Encode()
		activitiesWithKeys = append(activitiesWithKeys, ActivityWithKey{Activity: activities[i], EncodedKey: encodedKey})
	}

	wr.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := wr.MakeTemplateData(map[string]interface{}{
		"Events":             allEvents,
		"VenueMap":           venueMap,
		"VenueEncodedKeyMap": venueEncodedKeyMap,
		"BuildingOrder":      buildingInts,
		"BuildingKeyMap":     buildingKeyMap,
		"BuildingRoomMap":    buildingRoomMap,
		"RsvpStatuses":       GetAllRsvpStatuses(),
		"ActivitiesWithKeys": activitiesWithKeys,
	})

	functionMap := template.FuncMap{
		"makeLoginUrl":   makeLoginUrl,
		"dereferenceKey": func(key *datastore.Key) datastore.Key { return *key },
	}
	tpl := template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/main.html", "templates/events.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "events.html", data); err != nil {
		log.Errorf(ctx, "%v", err)
	}
}

func handleCreateUpdateEvent(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	wr.Request.ParseForm()
	form := wr.Request.Form

	for key, value := range form {
		b := new(bytes.Buffer)
		_, _ = fmt.Fprintf(b, "%s=\"%s\"\n", key, value)
		log.Infof(ctx, b.String())
	}

	event := Event{}
	venue, err := datastore.DecodeKey(form["venue"][0])
	if err != nil {
		log.Infof(ctx, "%v", err)
	}
	event.Name = form["name"][0]
	event.ShortName = form["shortName"][0]
	event.Venue = venue

	layout := "01/02/2006"
	event.StartDate, _ = time.Parse(layout, form["startDate"][0])
	event.EndDate, _ = time.Parse(layout, form["endDate"][0])

	allRsvpStatuses := GetAllRsvpStatuses()
	var statusesForEvent []RsvpStatus
	for _, statusIntStr := range form["rsvpStatus"] {
		statusInt, _ := strconv.ParseInt(statusIntStr, 10, 64)
		statusInfo := allRsvpStatuses[statusInt]
		statusesForEvent = append(statusesForEvent, statusInfo.Status)
	}
	event.RsvpStatuses = statusesForEvent

	var rooms []*datastore.Key
	for _, room := range form["rooms"] {

		components := strings.Split(room, "_")
		log.Infof(ctx, "found room in building "+components[0]+" with number "+components[1])
		q := datastore.NewQuery("Building").Filter("Code =", components[0]).KeysOnly()
		buildingKeys, _ := q.GetAll(ctx, nil)
		log.Infof(ctx, "Found building keys: %v", buildingKeys)
		//q = datastore.NewQuery("Room").Ancestor(buildingKeys[0])
		roomNumber, _ := strconv.ParseInt(components[1], 10, 64)
		log.Infof(ctx, "Room number: %v", roomNumber)
		q = datastore.NewQuery("Room").Filter("Building =", buildingKeys[0]).KeysOnly() //.Filter("RoomNumber =", roomNumber).KeysOnly()
		roomKeys, _ := q.GetAll(ctx, nil)
		log.Infof(ctx, "room keys: %v", roomKeys)

		rooms = append(rooms, roomKeys[0])
	}
	event.Rooms = rooms

	var activityKeys []*datastore.Key
	for _, encodedActivityKey := range form["activity"] {
		activityKey, _ := datastore.DecodeKey(encodedActivityKey)
		activityKeys = append(activityKeys, activityKey)
	}
	event.Activities = activityKeys

	var current = (form["current"] != nil && len(form["current"]) > 0 && form["current"][0] == "on")
	if current {
		var allEvents []*Event
		q := datastore.NewQuery("Event")
		eventKeys, _ := q.GetAll(ctx, &allEvents)

		for i, event := range allEvents {
			event.Current = false
			_, _ = datastore.Put(ctx, eventKeys[i], event)
		}
	}

	event.Current = current
	EventKey := datastore.NewIncompleteKey(ctx, "Event", nil)
	datastore.Put(ctx, EventKey, &event)

	log.Infof(ctx, "event: %v", event)

}
