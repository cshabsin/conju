package conju

// TODO: move to "package models"?

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cshabsin/conju/activity"
	"github.com/cshabsin/conju/invitation"
	"github.com/cshabsin/conju/model/event"
	"github.com/cshabsin/conju/model/housing"
	"github.com/cshabsin/conju/model/venue"
	"google.golang.org/appengine/datastore"
)

type CurrentEvent struct {
	Key *datastore.Key
}

// Sets up Event in the WrappedRequest.
func EventGetter(ctx context.Context, wr *WrappedRequest) error {
	if wr.hasRunEventGetter {
		return nil // Only retrieve once.
	}
	wr.hasRunEventGetter = true
	var key *datastore.Key
	found, err := event.GetEventForHost(ctx, wr.Host, &wr.Event, &key)
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

	if key != nil {
		ev, err := event.GetEvent(ctx, key)
		if err == nil {
			// We have retrieved the event successfully.
			wr.Event = ev
			wr.EventKey = key
			wr.TemplateData["CurrentEvent"] = ev
			return nil
		}
		// eat the error and fall back to the db current event
	}

	ev, err := event.GetCurrentEvent(ctx)
	if err != nil {
		return err
	}
	wr.Event = ev

	wr.TemplateData["CurrentEvent"] = wr.Event
	wr.EventKey = ev.Key
	wr.SetSessionValue("EventKey", ev.Key.Encode())
	wr.SaveSession()

	return nil
}

func handleEvents(ctx context.Context, wr WrappedRequest) {
	tic := time.Now()

	allEvents, err := event.GetAllEvents(ctx)
	if err != nil {
		http.Error(wr.ResponseWriter, err.Error(), http.StatusInternalServerError)
		log.Printf("GetAllEvents: %v", err)
		return
	}

	log.Printf("GetAllEvents: Datastore lookup took %s", time.Since(tic).String())
	log.Printf("Rendering %d events", len(allEvents))

	allVenues, err := venue.AllVenues(ctx)
	if err != nil {
		http.Error(wr.ResponseWriter, err.Error(), http.StatusInternalServerError)
		log.Printf("AllVenues: %v", err)
		return
	}

	// fetch this with an ajax call eventually
	var buildings []*housing.Building
	var buildingOrder []int64
	var rooms []*housing.Room
	roomMap := make(map[int64]*housing.Room)
	buildingRoomMap := make(map[int64][]*housing.Room)
	buildingKeyMap := make(map[int64]*housing.Building)
	if len(allVenues) == 1 {
		q := datastore.NewQuery("Building").Ancestor(allVenues[0].Key).Order("Name")
		buildingKeys, _ := q.GetAll(ctx, &buildings)

		for i, building := range buildings {
			buildingOrder = append(buildingOrder, buildingKeys[i].IntID())
			buildingKeyMap[buildingKeys[i].IntID()] = building
		}

		// whoops this query doesn't use venue
		q = datastore.NewQuery("Room").Order("RoomNumber").Order("Partition")
		roomKeys, _ := q.GetAll(ctx, &rooms)

		for j, room := range rooms {
			buildingRoomMap[room.Building.IntID()] = append(buildingRoomMap[room.Building.IntID()], room)
			roomMap[roomKeys[j].IntID()] = room
		}
	}

	activitiesWithKeys, err := activity.QueryAll(ctx)
	if err != nil {
		log.Printf("activity.QueryAll: %v", err)
	}

	err = wr.Request.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v", err)
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
			log.Printf("Error decoding key from editEvent: %v", err)
		}
	}
	log.Print(2)
	var editEvent *event.Event
	eventRoomMap := make(map[string]bool)
	rsvpStatusMap := make(map[invitation.RsvpStatus]bool)
	activityMap := make(map[string]bool)
	if editEventKey != nil {
		var err error
		editEvent, err = event.GetEvent(ctx, editEventKey)
		if err != nil {
			log.Printf("Get event: %v", err)
		}
		for _, roomKey := range editEvent.Rooms {
			room := roomMap[roomKey.IntID()]
			building := buildingKeyMap[room.Building.IntID()]
			eventRoomMap[building.Code+"_"+strconv.Itoa(room.RoomNumber)+"_"+room.Partition] = true
		}
		for _, status := range editEvent.RsvpStatuses {
			rsvpStatusMap[status] = true
		}
		for _, activityKey := range editEvent.Activities {
			activityMap[activityKey.Encode()] = true
		}
	}

	wr.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	log.Print(3)

	data := wr.MakeTemplateData(map[string]interface{}{
		"Events":              allEvents,
		"BuildingOrder":       buildingOrder,
		"BuildingKeyMap":      buildingKeyMap,
		"BuildingRoomMap":     buildingRoomMap,
		"RsvpStatuses":        invitation.GetAllRsvpStatuses(),
		"ActivitiesWithKeys":  activitiesWithKeys,
		"EditEvent":           editEvent,
		"EditEventKeyEncoded": editEventKeyEncoded,
		"RsvpStatusMap":       rsvpStatusMap,
		"ActivityMap":         activityMap,
		"RoomMap":             eventRoomMap,
		"Venues":              allVenues,
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
		log.Printf("%v", err)
	}
	log.Print(4)
}

func handleCreateUpdateEvent(ctx context.Context, wr WrappedRequest) {
	wr.Request.ParseForm()
	form := wr.Request.Form

	for key, value := range form {
		log.Printf("%s=\"%s\"\n", key, value)
	}

	ev := &event.Event{}
	eventKey := datastore.NewIncompleteKey(ctx, "Event", nil)
	if form["editEventKeyEncoded"] != nil && form["editEventKeyEncoded"][0] != "" {
		var err error
		eventKey, err = datastore.DecodeKey(form["editEventKeyEncoded"][0])
		if err != nil {
			log.Printf("decoding event key from form: %v", err)
		}
		ev, err = event.GetEvent(ctx, eventKey)
		if err != nil {
			log.Printf("getting event from form: %v", err)
		}
	}

	venueKey, err := datastore.DecodeKey(form["venue"][0])
	if err != nil {
		log.Printf("decoding venue key from form: %v", err)
	}
	ev.Name = form["name"][0]
	ev.ShortName = form["shortName"][0]
	ev.SetVenueKey(venueKey)

	layout := "01/02/2006"
	ev.StartDate, err = time.Parse(layout, form["startDate"][0])
	if err != nil {
		log.Printf("decoding start date from form: %v", err)
	}
	ev.EndDate, err = time.Parse(layout, form["endDate"][0])
	if err != nil {
		log.Printf("decoding end date from form: %v", err)
	}

	allRsvpStatuses := invitation.GetAllRsvpStatuses()
	var statusesForEvent []invitation.RsvpStatus
	for _, statusIntStr := range form["rsvpStatus"] {
		statusInt, _ := strconv.ParseInt(statusIntStr, 10, 64)
		statusInfo := allRsvpStatuses[statusInt]
		statusesForEvent = append(statusesForEvent, statusInfo.Status)
	}
	ev.RsvpStatuses = statusesForEvent

	var rooms []*datastore.Key
	for _, room := range form["rooms"] {

		components := strings.Split(room, "_")
		//log.Printf( "found room in building "+components[0]+" with number "+components[1])
		q := datastore.NewQuery("Building").Filter("Code =", components[0]).KeysOnly()
		buildingKeys, err := q.GetAll(ctx, nil)
		if err != nil {
			log.Printf("Getting buildings by code %q: %v", components[0], err)
		}
		//log.Printf( "Found building keys: %v", buildingKeys)
		roomNumber, err := strconv.ParseInt(components[1], 10, 64)
		if err != nil {
			log.Printf("Parsing value %q: %v", components[1], err)
		}
		//log.Printf( "Room number: %v", roomNumber)
		q = datastore.NewQuery("Room").Filter("Building =", buildingKeys[0]).Filter("RoomNumber =", roomNumber).Filter("Partition =", components[2]).KeysOnly()
		roomKeys, err := q.GetAll(ctx, nil)
		if err != nil {
			log.Printf("Reading room: %v", err)
		}
		//log.Printf( "room keys: %v", roomKeys)

		rooms = append(rooms, roomKeys[0])
	}
	ev.Rooms = rooms

	var activityKeys []*datastore.Key
	for _, encodedActivityKey := range form["activity"] {
		activityKey, _ := datastore.DecodeKey(encodedActivityKey)
		activityKeys = append(activityKeys, activityKey)
	}
	ev.Activities = activityKeys

	makeCurrent := (form["current"] != nil && len(form["current"]) > 0 && form["current"][0] == "on")
	if makeCurrent {
		// make all events not current
		allEvents, err := event.GetAllEvents(ctx)
		if err != nil {
			log.Printf("GetAllEvents: %v", err)
		}

		for _, event := range allEvents {
			event.Current = false
			_, err := datastore.Put(ctx, event.Key, event)
			if err != nil {
				log.Printf("Updating event: %v", err)
			}
		}
	}

	ev.Current = makeCurrent
	if err := event.PutEvent(ctx, ev); err != nil {
		log.Printf("PutEvent: %v", err)
		http.Error(wr.ResponseWriter, fmt.Sprintf("PutEvent: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(wr.ResponseWriter, wr.Request, "events", http.StatusSeeOther)
}
