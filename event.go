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

func getEventForHost(wr *WrappedRequest, e **Event, key **datastore.Key) (bool, error) {
	host := wr.GetHost()
	// TODO: generalize this for multiple hostnames/events.
	if host != "psr2019.shabsin.com" {
		return false, nil
	}
	shortname := "PSR2019"

	var keys []*datastore.Key
	var events []*Event
	q := datastore.NewQuery("Event").Filter("ShortName =", shortname)
	keys, err := q.GetAll(wr.Context, &events)
	if err != nil {
		log.Errorf(wr.Context, "Error querying for %s(url) event: %v", shortname, err)
		return false, nil
	}
	if len(keys) == 0 {
		log.Errorf(wr.Context, "Found no %s(url) event", shortname)
		return false, nil
	}
	if len(keys) > 1 {
		log.Errorf(wr.Context, "Found more than one %s(url) event (%d)", shortname, len(keys))
		return false, nil
	}
	*e = events[0]
	*key = keys[0]
	return true, nil
}

// Sets up Event in the WrappedRequest.
func EventGetter(wr *WrappedRequest) error {
	if wr.hasRunEventGetter {
		return nil // Only retrieve once.
	}
	wr.hasRunEventGetter = true
	var key *datastore.Key
	found, err := getEventForHost(wr, &wr.Event, &key)
	if err != nil {
		return err
	}
	if found {
		wr.TemplateData["CurrentEvent"] = wr.Event
		wr.EventKey = key
		return nil
	}

	key, err = wr.RetrieveKeyFromSession("EventKey")
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
	var allEventsEncodedKeys []string
	eventKeys, err := q.GetAll(ctx, &allEvents)
	if err != nil {
		http.Error(wr.ResponseWriter, err.Error(), http.StatusInternalServerError)
		log.Errorf(ctx, "GetAll: %v", err)
		return
	}
	for ev := 0; ev < len(eventKeys); ev++ {
		allEventsEncodedKeys = append(allEventsEncodedKeys, eventKeys[ev].Encode())
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
	roomMap := make(map[int64]Room)
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

		// whoops this query doesn't use venue
		q = datastore.NewQuery("Room").Order("RoomNumber").Order("Partition")
		roomKeys, _ := q.GetAll(ctx, &rooms)

		for j := 0; j < len(rooms); j++ {
			//building := buildingKeyMap[rooms[j].Building.IntID()]
			//log.Infof(ctx, "Found room %d for building %s", rooms[j].RoomNumber, building.Name)
			roomList := buildingRoomMap[rooms[j].Building.IntID()]
			roomList = append(roomList, *(rooms[j]))
			buildingRoomMap[rooms[j].Building.IntID()] = roomList
			//log.Infof(ctx, "room list size: %d", len(roomList))
			roomMap[(*roomKeys[j]).IntID()] = *rooms[j]
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

	err = wr.Request.ParseForm()
	if err != nil {
		log.Errorf(wr.Context, "Error parsing form: %v", err)
	}
	setCurrentKeyEncoded := wr.Request.Form.Get("setCurrent")
	if setCurrentKeyEncoded != "" {
		wr.SetSessionValue("EventKey", setCurrentKeyEncoded)
		wr.SaveSession()
	}
	editEventKeyEncoded := wr.Request.Form.Get("editEvent")
	var editEventKey *datastore.Key
	if editEventKeyEncoded != "" {
		editEventKey, err = datastore.DecodeKey(editEventKeyEncoded)
		if err != nil {
			log.Errorf(wr.Context, "Error decoding key from editEvent: %v", err)
		}
	}
	var editEvent Event
	err = datastore.Get(wr.Context, editEventKey, &editEvent)

	eventRoomMap := make(map[string]bool)
	rsvpStatusMap := make(map[int]bool)
	activityMap := make(map[string]bool)
	if editEventKey != nil {
		for _, roomKey := range editEvent.Rooms {
			room := roomMap[roomKey.IntID()]
			building := buildingKeyMap[room.Building.IntID()]
			eventRoomMap[building.Code+"_"+strconv.Itoa(room.RoomNumber)] = true
		}
		for _, status := range editEvent.RsvpStatuses {
			rsvpStatusMap[int(status)] = true
		}
		for _, activityKey := range editEvent.Activities {
			activityMap[activityKey.Encode()] = true
		}
	}

	wr.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := wr.MakeTemplateData(map[string]interface{}{
		"Events":              allEvents,
		"EventKeys":           allEventsEncodedKeys,
		"VenueMap":            venueMap,
		"VenueEncodedKeyMap":  venueEncodedKeyMap,
		"BuildingOrder":       buildingInts,
		"BuildingKeyMap":      buildingKeyMap,
		"BuildingRoomMap":     buildingRoomMap,
		"RsvpStatuses":        GetAllRsvpStatuses(),
		"ActivitiesWithKeys":  activitiesWithKeys,
		"EditEvent":           editEvent,
		"EditEventKeyEncoded": editEventKeyEncoded,
		"RsvpStatusMap":       rsvpStatusMap,
		"ActivityMap":         activityMap,
		"RoomMap":             eventRoomMap,
	})

	functionMap := template.FuncMap{
		"makeLoginUrl":   makeLoginUrl,
		"dereferenceKey": func(key *datastore.Key) datastore.Key { return *key },
		"encodeKey": func(key *datastore.Key) string {
			if key == nil {
				return ""
			}
			return key.Encode()
		},
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
	eventKey := datastore.NewIncompleteKey(ctx, "Event", nil)
	if form["editEventKeyEncoded"] != nil && form["editEventKeyEncoded"][0] != "" {
		eventKey, _ = datastore.DecodeKey(form["editEventKeyEncoded"][0])
		_ = datastore.Get(wr.Context, eventKey, &event)
	}

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
		//log.Infof(ctx, "found room in building "+components[0]+" with number "+components[1])
		q := datastore.NewQuery("Building").Filter("Code =", components[0]).KeysOnly()
		buildingKeys, _ := q.GetAll(ctx, nil)
		//log.Infof(ctx, "Found building keys: %v", buildingKeys)
		roomNumber, _ := strconv.ParseInt(components[1], 10, 64)
		//log.Infof(ctx, "Room number: %v", roomNumber)
		q = datastore.NewQuery("Room").Filter("Building =", buildingKeys[0]).Filter("RoomNumber =", roomNumber).KeysOnly()
		roomKeys, _ := q.GetAll(ctx, nil)
		//log.Infof(ctx, "room keys: %v", roomKeys)

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
	datastore.Put(ctx, eventKey, &event)

	log.Infof(ctx, "event: %v", event)
	http.Redirect(wr.ResponseWriter, wr.Request, "events", http.StatusSeeOther)
}
