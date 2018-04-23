package conju

import (
	"html/template"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

func handleRoomingTool(wr WrappedRequest) {
	ctx := appengine.NewContext(wr.Request)
	var invitations []*Invitation

	q := datastore.NewQuery("Invitation").Filter("Event =", wr.EventKey)
	_, err := q.GetAll(ctx, &invitations)
	if err != nil {
		log.Errorf(ctx, "fetching invitations: %v", err)
	}

	rsvpToGroupsMap := make(map[RsvpStatus][][]Person)
	for _, invitation := range invitations {
		rsvpMap := invitation.ClusterByRsvp(ctx)
		for k, v := range rsvpMap {
			if GetAllRsvpStatuses()[k].Attending {
				if listForRsvp, present := rsvpToGroupsMap[k]; present {
					listForRsvp = append(listForRsvp, v)
					rsvpToGroupsMap[k] = listForRsvp
				} else {
					listForRsvp = [][]Person{}
					listForRsvp = append(listForRsvp, v)
					rsvpToGroupsMap[k] = listForRsvp
				}
			}
		}
	}

	buildingsMap := make(map[datastore.Key]Building)
	var buildings []Building
	q = datastore.NewQuery("Building") // .Filter("Venue =", wr.Event.Venue)
	keys, err := q.GetAll(ctx, &buildings)
	for i, buildingKey := range keys {
		log.Infof(ctx, "%s -> %v", buildings[i].Name, buildings[i])
		buildingsMap[*buildingKey] = buildings[i]
	}

	var buildingsInOrder []Building
	var availableRooms []RealRoom
	var buildingsToRooms = make(map[Building][]RealRoom)

	for i, room := range wr.Event.Rooms {
		var rm Room
		datastore.Get(ctx, room, &rm)
		buildingKey := rm.Building
		building := buildingsMap[*buildingKey]
		if i == 0 || buildingsInOrder[len(buildingsInOrder)-1] != building {
			buildingsInOrder = append(buildingsInOrder, building)
		}

		bedstring := ""
		for _, bed := range rm.Beds {
			switch bed {
			case King:
				bedstring += "K"
			case Queen:
				bedstring += "Q"
			case Double:
				bedstring += "D"
			case Twin:
				bedstring += "T"
			case Cot:
				bedstring += "C"

			}
		}
		realRoom := RealRoom{
			Room:       rm,
			Building:   buildingsMap[*buildingKey],
			BedsString: bedstring,
		}

		roomsForBuilding := buildingsToRooms[building]

		if roomsForBuilding == nil {
			roomsForBuilding = make([]RealRoom, 0)

		}
		roomsForBuilding = append(roomsForBuilding, realRoom)
		buildingsToRooms[building] = roomsForBuilding

		availableRooms = append(availableRooms, realRoom)
	}

	statusOrder := []RsvpStatus{ThuFriSat, FriSat}
	tpl := template.Must(template.New("").ParseFiles("templates/main.html", "templates/roomingTool.html"))
	data := wr.MakeTemplateData(map[string]interface{}{
		"RsvpToGroupsMap":  rsvpToGroupsMap,
		"StatusOrder":      statusOrder,
		"AllRsvpStatuses":  GetAllRsvpStatuses(),
		"AvailableRooms":   availableRooms,
		"BuildingsToRooms": buildingsToRooms,
		"BuildingsInOrder": buildingsInOrder,
	})
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "roomingTool.html", data); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}

}
