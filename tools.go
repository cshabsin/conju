package conju

import (
	"html/template"
	"sort"

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

	statusOrder := []RsvpStatus{ThuFriSat, FriSat, Maybe}
	rsvpToGroupsMap := make(map[RsvpStatus][][]Person)
	var noRsvps [][]Person
	for _, invitation := range invitations {
		rsvpMap, noResponse := invitation.ClusterByRsvp(ctx)
		for _, s := range statusOrder {
			if peopleForRsvp, pr := rsvpMap[s]; pr {
				if listForRsvp, present := rsvpToGroupsMap[s]; present {
					listForRsvp = append(listForRsvp, peopleForRsvp)
					rsvpToGroupsMap[s] = listForRsvp
				} else {
					listForRsvp = [][]Person{}
					listForRsvp = append(listForRsvp, peopleForRsvp)
					rsvpToGroupsMap[s] = listForRsvp
				}
			}
		}
		if len(noResponse) > 0 {
			noRsvps = append(noRsvps, noResponse)
		}
	}

	for _, v := range rsvpToGroupsMap {
		sort.Slice(v, func(a, b int) bool { return SortByFirstName(v[a][0], v[b][0]) })
	}

	sort.Slice(noRsvps, func(a, b int) bool { return SortByFirstName(noRsvps[a][0], noRsvps[b][0]) })

	buildingsMap := make(map[datastore.Key]Building)
	var buildings []Building
	q = datastore.NewQuery("Building") // .Filter("Venue =", wr.Event.Venue)
	keys, err := q.GetAll(ctx, &buildings)
	for i, buildingKey := range keys {
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

	tpl := template.Must(template.New("").ParseFiles("templates/main.html", "templates/roomingTool.html"))
	data := wr.MakeTemplateData(map[string]interface{}{
		"RsvpToGroupsMap":  rsvpToGroupsMap,
		"NoRsvps":          noRsvps,
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
