package activity

import (
	"context"

	"google.golang.org/appengine/datastore"
)

type Activity struct {
	Keyword     string
	Description string
	NeedsLeader bool
}

type ActivityWithKey struct {
	EncodedKey string
	Activity   Activity
}

func QueryAll(ctx context.Context) ([]ActivityWithKey, error) {
	var activities []Activity
	q := datastore.NewQuery("Activity").Order("Keyword")
	activityKeys, err := q.GetAll(ctx, &activities)
	if err != nil {
		return nil, err
	}

	var activitiesWithKeys []ActivityWithKey
	for i, activityKey := range activityKeys {
		encodedKey := activityKey.Encode()
		activitiesWithKeys = append(activitiesWithKeys,
			ActivityWithKey{Activity: activities[i], EncodedKey: encodedKey})
	}

	return activitiesWithKeys, nil
}

func Realize(ctx context.Context, activityKeys []*datastore.Key) ([]*Activity, error) {
	activities := make([]*Activity, len(activityKeys))
	if err := datastore.GetMulti(ctx, activityKeys, activities); err != nil {
		return nil, err
	}
	return activities, nil

	// var realActivities []Activity
	// for _, activity := range activities {
	// 	realActivities = append(realActivities, *activity)
	// }
	// return realActivities, nil
}
