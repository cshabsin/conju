package venue

import (
	"context"
	"log"

	"google.golang.org/appengine/datastore"
)

type venueDB struct {
	Name          string
	ShortName     string
	ContactPerson string
	ContactPhone  string
	ContactEmail  string
	Website       string
}

type Venue struct {
	Key           *datastore.Key
	Name          string
	ShortName     string
	ContactPerson string
	ContactPhone  string
	ContactEmail  string
	Website       string
}

func fromDB(ctx context.Context, key *datastore.Key, v *venueDB) (*Venue, error) {
	if v == nil {
		v = new(venueDB)
		err := datastore.Get(ctx, key, v)
		if err != nil {
			log.Printf("error loading venue from db for key %v: %v", key.Encode(), err)
			return nil, err
		}
	}
	return &Venue{
		Key:           key,
		Name:          v.Name,
		ShortName:     v.ShortName,
		ContactPerson: v.ContactPerson,
		ContactPhone:  v.ContactPhone,
		ContactEmail:  v.ContactEmail,
		Website:       v.Website,
	}, nil
}

func FromKey(ctx context.Context, key *datastore.Key) (*Venue, error) {
	return fromDB(ctx, key, nil)
}

func AllVenues(ctx context.Context) ([]*Venue, error) {
	var venueData []*venueDB
	q := datastore.NewQuery("Venue")
	keys, err := q.GetAll(ctx, &venueData)
	if err != nil {
		return nil, err
	}
	var venues []*Venue
	for i, vdb := range venueData {
		venue, err := fromDB(ctx, keys[i], vdb)
		if err != nil {
			return nil, err
		}
		venues = append(venues, venue)
	}
	return venues, nil
}
