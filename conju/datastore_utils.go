package conju

import (
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/datastore"
	"github.com/gin-gonic/gin"

	"github.com/cshabsin/conju/conju/dsclient"
	"github.com/cshabsin/conju/conju/login"
	"github.com/cshabsin/conju/model/person"
)

func ClearAllData(c *gin.Context) {
	wr, ok := c.MustGet("wrappedRequest").(*WrappedRequest)
	if !ok {
		log.Printf("could not get wrapped request from context")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	ctx := c.Request.Context()

	c.String(http.StatusOK, "Disabled for now.\n")
	wr.Values["event"] = nil
	wr.SaveSession()

	//entityNames := []string{"Activity", "Event", "CurrentEvent", "Person", "Invitation", "LoginCode", "Venue", "Building", "Room"}

	entityNames := []string{}
	for _, entityName := range entityNames {
		c.String(http.StatusOK, "Clearing: %s\n", entityName)
		q := datastore.NewQuery(entityName).KeysOnly()

		keys, err := dsclient.FromContext(ctx).GetAll(ctx, q, nil)
		if err != nil {
			log.Println("ClearAllData GetAll:", err)
			return
		}

		c.String(http.StatusOK, "	%d %s to delete\n", len(keys), entityName)

		err = dsclient.FromContext(ctx).DeleteMulti(ctx, keys)
		if err != nil {
			log.Println("ClearAllData DeleteMulti:", err)
			return
		}
	}
}

func RepairData(c *gin.Context) {
	ctx := c.Request.Context()
	q := datastore.NewQuery("Person")
	var people []person.Person
	personKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &people)
	if err != nil {
		log.Printf("RepairData personQuery: %v", err)
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	for i := range personKeys {
		if people[i].LoginCode == "" {
			people[i].LoginCode = login.RandomLoginCodeString()
			_, err = dsclient.FromContext(ctx).Put(ctx, personKeys[i], &people[i])
			if err != nil {
				log.Printf("RepairData put(%s): %v", people[i].Email, err)
				c.String(http.StatusInternalServerError, fmt.Sprintf("put(%s): %v", people[i].Email, err))
				return
			}
		}
	}
	c.String(http.StatusOK, "Done.")
}
