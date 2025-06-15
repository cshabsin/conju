package activity

import (
	"context"

	"cloud.google.com/go/datastore"
	"github.com/cshabsin/conju/conju/dsclient"
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
	activityKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &activities)
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
	if err := dsclient.FromContext(ctx).GetMulti(ctx, activityKeys, activities); err != nil {
		return nil, err
	}
	return activities, nil

	// var realActivities []Activity
	// for _, activity := range activities {
	// 	realActivities = append(realActivities, *activity)
	// }
	// return realActivities, nil
}
