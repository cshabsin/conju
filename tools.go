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

	statusOrder := []RsvpStatus{ThuFriSat, FriSat}
	tpl := template.Must(template.New("").ParseFiles("templates/main.html", "templates/roomingTool.html"))
	data := wr.MakeTemplateData(map[string]interface{}{
		"RsvpToGroupsMap": rsvpToGroupsMap,
		"StatusOrder":     statusOrder,
		"AllRsvpStatuses": GetAllRsvpStatuses(),
	})
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "roomingTool.html", data); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}

}
